// Калибровка порогов MatchSimLow/MatchSimHigh для матчинга топиков к сюжетам.
//
// Реплеит матчинг на исторических данных storylines: каждый сюжет дня N
// выступает "кандидатом" (его title+state — ровно тот текст, который стадия E
// эмбеддила doc-моделью), пул для матчинга — сюжеты того же канала с
// first_seen < N. Считает два вида близости:
//   - query-doc: queryEmbed(кандидат) x сохранённый doc-эмбеддинг (как в проде);
//   - doc-doc: сохранённый эмбеддинг кандидата x сохранённый эмбеддинг сюжета.
//
// Пары кандидат-сюжет из top-K размечает LLM-судьёй (ConfirmMatch с одним
// вариантом): "тот же сюжет" или нет. По разметке печатает precision/recall
// по сетке порогов для обеих схем.
//
// Эмбеддинги и вердикты LLM кэшируются в JSON-файл, повторный запуск бесплатен.
//
// Запуск:
//
//	DATABASE_URL=postgres://... go run ./scripts/threshold_calibration \
//	  [-cache /tmp/calib_cache.json] [-pairs-csv /tmp/calib_pairs.csv] [-topk 5]
package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/Ra1ze505/goNewsBot/src/repository"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	log "github.com/sirupsen/logrus"
)

const (
	// Пары ниже обоих флоров считаем заведомо "не тот же сюжет" без LLM.
	labelFloorQuery = 0.30
	labelFloorDoc   = 0.55
	llmWorkers      = 4
)

type storyline struct {
	ID        int64
	ChannelID int64
	Title     string
	State     string
	FirstSeen time.Time
	Embedding []float32
}

type pair struct {
	CandID   int64
	PriorID  int64
	QuerySim float64
	DocSim   float64
	Match    bool
	Source   string // llm_yes | llm_no | assumed_no
}

type cache struct {
	QueryEmbeds map[string][]float32 `json:"query_embeds"` // storyline id -> query-эмбеддинг title+state
	Labels      map[string]bool      `json:"labels"`       // "candID->priorID" -> тот же сюжет
}

func main() {
	_ = godotenv.Load()

	cachePath := flag.String("cache", "/tmp/calib_cache.json", "файл кэша эмбеддингов и LLM-вердиктов")
	pairsCSV := flag.String("pairs-csv", "/tmp/calib_pairs.csv", "куда писать размеченные пары")
	topK := flag.Int("topk", 5, "сколько ближайших сюжетов рассматривать на кандидата")
	flag.Parse()

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ml, err := repository.NewMLRepository()
	if err != nil {
		log.Fatalf("failed to init ML repository: %v", err)
	}

	storylines, err := loadStorylines(db)
	if err != nil {
		log.Fatalf("failed to load storylines: %v", err)
	}
	log.Infof("Loaded %d storylines", len(storylines))

	c := loadCache(*cachePath)

	// Query-эмбеддинги всех кандидатов (с кэшем).
	if err := ensureQueryEmbeds(ml, storylines, c, *cachePath); err != nil {
		log.Fatalf("failed to embed candidates: %v", err)
	}

	pairs := buildPairs(storylines, c, *topK)
	log.Infof("Built %d candidate-prior pairs (top-%d union by query-doc and doc-doc)", len(pairs), *topK)

	if err := labelPairs(ml, storylines, pairs, c, *cachePath); err != nil {
		log.Fatalf("failed to label pairs: %v", err)
	}

	if err := writePairsCSV(*pairsCSV, storylines, pairs); err != nil {
		log.Fatalf("failed to write pairs csv: %v", err)
	}
	log.Infof("Labeled pairs written to %s", *pairsCSV)

	printReport(pairs)
}

func loadStorylines(db *sql.DB) ([]storyline, error) {
	rows, err := db.Query(`
		SELECT id, channel_id, title, state, first_seen, embedding
		FROM storylines
		WHERE embedding IS NOT NULL
		ORDER BY channel_id, first_seen, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []storyline
	for rows.Next() {
		var s storyline
		var vec pgvector.Vector
		if err := rows.Scan(&s.ID, &s.ChannelID, &s.Title, &s.State, &s.FirstSeen, &vec); err != nil {
			return nil, err
		}
		s.Embedding = vec.Slice()
		result = append(result, s)
	}
	return result, rows.Err()
}

func loadCache(path string) *cache {
	c := &cache{QueryEmbeds: map[string][]float32{}, Labels: map[string]bool{}}
	data, err := os.ReadFile(path)
	if err != nil {
		return c
	}
	if err := json.Unmarshal(data, c); err != nil {
		log.Warnf("failed to parse cache %s, starting fresh: %v", path, err)
		return &cache{QueryEmbeds: map[string][]float32{}, Labels: map[string]bool{}}
	}
	if c.QueryEmbeds == nil {
		c.QueryEmbeds = map[string][]float32{}
	}
	if c.Labels == nil {
		c.Labels = map[string]bool{}
	}
	return c
}

func saveCache(path string, c *cache) {
	data, err := json.Marshal(c)
	if err != nil {
		log.Warnf("failed to marshal cache: %v", err)
		return
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		log.Warnf("failed to write cache: %v", err)
	}
}

func ensureQueryEmbeds(ml repository.MLRepositoryInterface, storylines []storyline, c *cache, cachePath string) error {
	missing := 0
	for _, s := range storylines {
		if _, ok := c.QueryEmbeds[strconv.FormatInt(s.ID, 10)]; !ok {
			missing++
		}
	}
	if missing == 0 {
		log.Info("All query embeddings found in cache")
		return nil
	}
	log.Infof("Embedding %d candidates with query model", missing)

	done := 0
	for _, s := range storylines {
		key := strconv.FormatInt(s.ID, 10)
		if _, ok := c.QueryEmbeds[key]; ok {
			continue
		}
		vecs, err := ml.EmbedQueries([]string{s.Title + "\n" + s.State})
		if err != nil {
			saveCache(cachePath, c)
			return fmt.Errorf("embed storyline %d: %w", s.ID, err)
		}
		c.QueryEmbeds[key] = vecs[0]
		done++
		if done%25 == 0 {
			log.Infof("Embedded %d/%d", done, missing)
			saveCache(cachePath, c)
		}
	}
	saveCache(cachePath, c)
	return nil
}

// buildPairs: для каждого кандидата берёт top-K прошлых сюжетов по query-doc
// и top-K по doc-doc, объединяет.
func buildPairs(storylines []storyline, c *cache, topK int) []pair {
	var pairs []pair
	for i := range storylines {
		cand := storylines[i]
		qvec, ok := c.QueryEmbeds[strconv.FormatInt(cand.ID, 10)]
		if !ok {
			continue
		}

		type scored struct {
			priorIdx int
			qsim     float64
			dsim     float64
		}
		var pool []scored
		for j := range storylines {
			prior := storylines[j]
			if prior.ChannelID != cand.ChannelID || !prior.FirstSeen.Before(cand.FirstSeen) {
				continue
			}
			pool = append(pool, scored{
				priorIdx: j,
				qsim:     cosine(qvec, prior.Embedding),
				dsim:     cosine(cand.Embedding, prior.Embedding),
			})
		}
		if len(pool) == 0 {
			continue
		}

		selected := map[int]scored{}
		sort.Slice(pool, func(a, b int) bool { return pool[a].qsim > pool[b].qsim })
		for k := 0; k < topK && k < len(pool); k++ {
			selected[pool[k].priorIdx] = pool[k]
		}
		sort.Slice(pool, func(a, b int) bool { return pool[a].dsim > pool[b].dsim })
		for k := 0; k < topK && k < len(pool); k++ {
			selected[pool[k].priorIdx] = pool[k]
		}

		for _, sc := range selected {
			pairs = append(pairs, pair{
				CandID:   cand.ID,
				PriorID:  storylines[sc.priorIdx].ID,
				QuerySim: sc.qsim,
				DocSim:   sc.dsim,
			})
		}
	}
	return pairs
}

func labelPairs(ml repository.MLRepositoryInterface, storylines []storyline, pairs []pair, c *cache, cachePath string) error {
	byID := make(map[int64]storyline, len(storylines))
	for _, s := range storylines {
		byID[s.ID] = s
	}

	var toLabel []int
	for i := range pairs {
		p := &pairs[i]
		key := fmt.Sprintf("%d->%d", p.CandID, p.PriorID)
		if match, ok := c.Labels[key]; ok {
			p.Match = match
			p.Source = map[bool]string{true: "llm_yes", false: "llm_no"}[match]
			continue
		}
		if p.QuerySim < labelFloorQuery && p.DocSim < labelFloorDoc {
			p.Match = false
			p.Source = "assumed_no"
			continue
		}
		toLabel = append(toLabel, i)
	}
	log.Infof("Pairs to label with LLM: %d (cached: %d)", len(toLabel), len(c.Labels))

	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		sem  = make(chan struct{}, llmWorkers)
		errs []error
		done int
	)
	for _, idx := range toLabel {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()

			p := &pairs[i]
			cand := byID[p.CandID]
			prior := byID[p.PriorID]

			matchedID, isNew, err := ml.ConfirmMatch(
				repository.CandidateTopic{Title: cand.Title, Summary: cand.State},
				[]repository.StorylineBrief{{
					ID:         prior.ID,
					Title:      prior.Title,
					State:      prior.State,
					LastSeen:   prior.FirstSeen.Format("2006-01-02"),
					Similarity: p.QuerySim,
				}},
			)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("label %d->%d: %w", p.CandID, p.PriorID, err))
				return
			}
			p.Match = !isNew && matchedID == prior.ID
			p.Source = map[bool]string{true: "llm_yes", false: "llm_no"}[p.Match]
			c.Labels[fmt.Sprintf("%d->%d", p.CandID, p.PriorID)] = p.Match
			done++
			if done%20 == 0 {
				log.Infof("Labeled %d/%d", done, len(toLabel))
				saveCache(cachePath, c)
			}
		}(idx)
	}
	wg.Wait()
	saveCache(cachePath, c)

	if len(errs) > 0 {
		return fmt.Errorf("%d labeling errors, first: %w", len(errs), errs[0])
	}
	return nil
}

func writePairsCSV(path string, storylines []storyline, pairs []pair) error {
	byID := make(map[int64]storyline, len(storylines))
	for _, s := range storylines {
		byID[s.ID] = s
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	_ = w.Write([]string{"cand_id", "prior_id", "channel_id", "cand_day", "prior_day", "query_sim", "doc_sim", "match", "source", "cand_title", "prior_title"})
	for _, p := range pairs {
		cand, prior := byID[p.CandID], byID[p.PriorID]
		_ = w.Write([]string{
			strconv.FormatInt(p.CandID, 10),
			strconv.FormatInt(p.PriorID, 10),
			strconv.FormatInt(cand.ChannelID, 10),
			cand.FirstSeen.Format("2006-01-02"),
			prior.FirstSeen.Format("2006-01-02"),
			fmt.Sprintf("%.4f", p.QuerySim),
			fmt.Sprintf("%.4f", p.DocSim),
			strconv.FormatBool(p.Match),
			p.Source,
			cand.Title,
			prior.Title,
		})
	}
	return nil
}

func printReport(pairs []pair) {
	for _, scheme := range []struct {
		name string
		sim  func(pair) float64
	}{
		{"query-doc (прод: EmbedQueries x stored doc)", func(p pair) float64 { return p.QuerySim }},
		{"doc-doc (альтернатива: EmbedDocuments x stored doc)", func(p pair) float64 { return p.DocSim }},
	} {
		var matchSims, nonMatchSims []float64
		for _, p := range pairs {
			if p.Match {
				matchSims = append(matchSims, scheme.sim(p))
			} else {
				nonMatchSims = append(nonMatchSims, scheme.sim(p))
			}
		}
		sort.Float64s(matchSims)
		sort.Float64s(nonMatchSims)

		fmt.Printf("\n=== Схема: %s ===\n", scheme.name)
		fmt.Printf("Пар 'тот же сюжет': %d, 'другой': %d\n", len(matchSims), len(nonMatchSims))
		fmt.Printf("Матчи:    p5=%.3f p25=%.3f median=%.3f p75=%.3f p95=%.3f\n",
			pct(matchSims, 5), pct(matchSims, 25), pct(matchSims, 50), pct(matchSims, 75), pct(matchSims, 95))
		fmt.Printf("Не-матчи: p5=%.3f p25=%.3f median=%.3f p75=%.3f p95=%.3f\n",
			pct(nonMatchSims, 5), pct(nonMatchSims, 25), pct(nonMatchSims, 50), pct(nonMatchSims, 75), pct(nonMatchSims, 95))

		fmt.Printf("%9s %14s %17s %10s %22s\n", "порог", "матчей >= t", "не-матчей >= t", "precision", "потеряно матчей < t")
		for t := 0.30; t <= 0.92; t += 0.02 {
			tp := countAtOrAbove(matchSims, t)
			fp := countAtOrAbove(nonMatchSims, t)
			precision := 0.0
			if tp+fp > 0 {
				precision = float64(tp) / float64(tp+fp)
			}
			lost := len(matchSims) - tp
			fmt.Printf("%9.2f %14d %17d %10.3f %12d (%5.1f%%)\n",
				t, tp, fp, precision, lost, 100*float64(lost)/math.Max(1, float64(len(matchSims))))
		}
	}
}

func pct(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(p / 100 * float64(len(sorted)-1))
	return sorted[idx]
}

func countAtOrAbove(sorted []float64, t float64) int {
	return len(sorted) - sort.SearchFloat64s(sorted, t)
}

func cosine(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

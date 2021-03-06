package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"runtime"
	"time"
	//"strconv"
	"encoding/gob"
	"math/rand"

	"github.com/ndaniels/esfragbag/bow"
	"github.com/ndaniels/esfragbag/bowdb"
	"github.com/ndaniels/tools/util"
)

type distType int

const (
	cosineDist    distType = iota
	euclideanDist          = iota
)

var (
	fragmentLibraryLoc  = ""
	pdbQuery            = ""
	metric              = cosineDist
	metricFlag          = ""
	potentialTargetsLoc = ""
	maxRadius           float64
	clusterRadius       float64
	lasttime            = time.Now().UTC().UnixNano()
	gobLoc              = ""
	json                = ""
	other               = ""
)

func init() {
	log.SetFlags(0)
	flag.StringVar(&fragmentLibraryLoc, "fragLib", fragmentLibraryLoc, "the location of the fragment library centers")
	flag.StringVar(&gobLoc, "clusters", gobLoc, "the location of the serialized clusters database")
	flag.StringVar(&pdbQuery, "pdbQuery", pdbQuery, "the search query library as a pdb")
	flag.StringVar(&json, "json", json, "json file")
	flag.StringVar(&metricFlag, "metricFlag", metricFlag, "Choice of metric to use; valid options are 'cosine' and 'euclidean'")
	// flag.StringVar(&potentialTargetsLoc, "potentialTargets", potentialTargetsLoc, "the location of the full fragment library database")
	flag.Float64Var(&maxRadius, "maxRadius", maxRadius, "maximum radius to search in")
	flag.Float64Var(&clusterRadius, "clusterRadius", clusterRadius, "maximum cluster radius in database")

	flag.Parse()

	if metricFlag == "cosine" {
		metric = cosineDist
	}
	if metricFlag == "euclidean" {
		metric = euclideanDist
	}
}

func newSearchResult(query, entry bow.Bowed) bowdb.SearchResult {
	return bowdb.SearchResult{
		Bowed:  entry,
		Cosine: query.Bow.Cosine(entry.Bow),
		Euclid: query.Bow.Euclid(entry.Bow),
	}
}

func timer() int64 {
	old := lasttime
	lasttime = time.Now().UTC().UnixNano()
	return lasttime - old
}

func dec_gob_ss_db(name string) [][]bow.Bowed {
	buf_bytes, err := ioutil.ReadFile(name)
	if err != nil {
		log.Fatal("Open file error:", err)
	}
	var buf bytes.Buffer
	buf.Write(buf_bytes)
	var db_slices [][]bow.Bowed
	dec := gob.NewDecoder(&buf)
	err = dec.Decode(&db_slices)
	if err != nil {
		log.Fatal("decode error:", err)
	}
	return db_slices
}

func main() {
	//start := time.Now()
	rand.Seed(1)
	//fmt.Println("Loading query")
	flagCpu := runtime.NumCPU()
	fragmentLib := util.Library(json)
	pdbQueries := make([]string, 1)
	pdbQueries[0] = pdbQuery
	bows := util.ProcessBowers(pdbQueries, fragmentLib, false, flagCpu, util.FlagQuiet)
	// for b := range bows {
	//   searchQuery.Add(b)
	// }

	db_centers, _ := bowdb.Open(fragmentLibraryLoc)
	db_centers.ReadAll()
	//fmt.Println(fmt.Sprintf("\t%d",timer()))

	//fmt.Println("Unserializing gob")
	db_slices := dec_gob_ss_db(gobLoc)
	var m map[string]int
	m = make(map[string]int)
	for i, center := range db_centers.Entries {
		m[center.Id] = i
	}
	//fmt.Println(fmt.Sprintf("\t%d",timer()))

	sortBy := bowdb.SortByEuclid
	if metric == cosineDist {
		sortBy = bowdb.SortByCosine
	}

	var coarse_search = bowdb.SearchOptions{
		Limit:  -1,
		Min:    0.0,
		Max:    (float64(clusterRadius) + float64(maxRadius)),
		SortBy: sortBy,
		Order:  bowdb.OrderAsc,
	}

	//var fine_search = bowdb.SearchOptions{
	//Limit:  -1,
	//Min:    0.0,
	//Max:    float64(maxRadius),
	//SortBy: bowdb.SortByEuclid,
	// Order:  bowdb.OrderAsc,
	//}

	//fmt.Println("Computing coarse results")
	for b := range bows {
		var coarse_results []bowdb.SearchResult
		coarse_results = db_centers.Search(coarse_search, b)
		//coarse_results_time := timer()
		//fmt.Println(fmt.Sprintf("\t%d",coarse_results_time))
		//fmt.Println(fmt.Sprintf("\tCount: %d",len(coarse_results)))

		// fmt.Println("Computing fine results")
		var fine_results []bowdb.SearchResult
		for _, center := range coarse_results {
			for _, entry := range db_slices[m[center.Id]] {
				var dist float64
				switch metric {
				case cosineDist:
					dist = b.Bow.Cosine(entry.Bow)
				case euclideanDist:
					dist = b.Bow.Euclid(entry.Bow)
				}
				if dist <= float64(maxRadius) {
					result := newSearchResult(b, entry)
					fmt.Printf(entry.Id)
					fmt.Printf(" ")
					fmt.Printf("%v", dist)
					fmt.Printf(" ")
					fine_results = append(fine_results, result)
				}
			}
		}
	}

}

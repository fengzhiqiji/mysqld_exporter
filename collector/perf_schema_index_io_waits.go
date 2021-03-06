// Scrape `performance_schema.table_io_waits_summary_by_index_usage`.

package collector

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

const perfIndexIOWaitsQuery = `
	SELECT OBJECT_SCHEMA, OBJECT_NAME, ifnull(INDEX_NAME, 'NONE') as INDEX_NAME,
	    COUNT_FETCH, COUNT_INSERT, COUNT_UPDATE, COUNT_DELETE,
	    SUM_TIMER_FETCH, SUM_TIMER_INSERT, SUM_TIMER_UPDATE, SUM_TIMER_DELETE
	  FROM performance_schema.table_io_waits_summary_by_index_usage
	  WHERE OBJECT_SCHEMA NOT IN ('mysql', 'performance_schema')
	`

// Metric descriptors.
var (
	performanceSchemaIndexWaitsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, performanceSchema, "index_io_waits_total"),
		"The total number of index I/O wait events for each index and operation.",
		[]string{"schema", "name", "index", "operation"}, nil,
	)
	performanceSchemaIndexWaitsTimeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, performanceSchema, "index_io_waits_seconds_total"),
		"The total time of index I/O wait events for each index and operation.",
		[]string{"schema", "name", "index", "operation"}, nil,
	)
)

// ScrapePerfIndexIOWaits collects for `performance_schema.table_io_waits_summary_by_index_usage`.
type ScrapePerfIndexIOWaits struct{}

// Name of the Scraper. Should be unique.
func (ScrapePerfIndexIOWaits) Name() string {
	return "perf_schema.indexiowaits"
}

// Help describes the role of the Scraper.
func (ScrapePerfIndexIOWaits) Help() string {
	return "Collect metrics from performance_schema.table_io_waits_summary_by_index_usage"
}

// Scrape collects data from database connection and sends it over channel as prometheus metric.
func (ScrapePerfIndexIOWaits) Scrape(db *sql.DB, ch chan<- prometheus.Metric) error {
	perfSchemaIndexWaitsRows, err := db.Query(perfIndexIOWaitsQuery)
	if err != nil {
		return err
	}
	defer perfSchemaIndexWaitsRows.Close()

	var (
		objectSchema, objectName, indexName               string
		countFetch, countInsert, countUpdate, countDelete uint64
		timeFetch, timeInsert, timeUpdate, timeDelete     uint64
	)

	for perfSchemaIndexWaitsRows.Next() {
		if err := perfSchemaIndexWaitsRows.Scan(
			&objectSchema, &objectName, &indexName,
			&countFetch, &countInsert, &countUpdate, &countDelete,
			&timeFetch, &timeInsert, &timeUpdate, &timeDelete,
		); err != nil {
			return err
		}
		ch <- prometheus.MustNewConstMetric(
			performanceSchemaIndexWaitsDesc, prometheus.CounterValue, float64(countFetch),
			objectSchema, objectName, indexName, "fetch",
		)
		// We only include the insert column when indexName is NONE.
		if indexName == "NONE" {
			ch <- prometheus.MustNewConstMetric(
				performanceSchemaIndexWaitsDesc, prometheus.CounterValue, float64(countInsert),
				objectSchema, objectName, indexName, "insert",
			)
		}
		ch <- prometheus.MustNewConstMetric(
			performanceSchemaIndexWaitsDesc, prometheus.CounterValue, float64(countUpdate),
			objectSchema, objectName, indexName, "update",
		)
		ch <- prometheus.MustNewConstMetric(
			performanceSchemaIndexWaitsDesc, prometheus.CounterValue, float64(countDelete),
			objectSchema, objectName, indexName, "delete",
		)
		ch <- prometheus.MustNewConstMetric(
			performanceSchemaIndexWaitsTimeDesc, prometheus.CounterValue, float64(timeFetch)/picoSeconds,
			objectSchema, objectName, indexName, "fetch",
		)
		// We only update write columns when indexName is NONE.
		if indexName == "NONE" {
			ch <- prometheus.MustNewConstMetric(
				performanceSchemaIndexWaitsTimeDesc, prometheus.CounterValue, float64(timeInsert)/picoSeconds,
				objectSchema, objectName, indexName, "insert",
			)
		}
		ch <- prometheus.MustNewConstMetric(
			performanceSchemaIndexWaitsTimeDesc, prometheus.CounterValue, float64(timeUpdate)/picoSeconds,
			objectSchema, objectName, indexName, "update",
		)
		ch <- prometheus.MustNewConstMetric(
			performanceSchemaIndexWaitsTimeDesc, prometheus.CounterValue, float64(timeDelete)/picoSeconds,
			objectSchema, objectName, indexName, "delete",
		)
	}
	return nil
}

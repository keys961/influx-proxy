// Copyright 2016 Eleme. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package backend

import (
	"errors"
	"github.com/influxdata/influxql"
)

var (
	ErrIllegalQL = errors.New("illegal InfluxQL")
)

func GetMeasurementsFromInfluxQL(q string) (m []string, err error) {
	stmt, err := influxql.ParseStatement(q)
	if err != nil {
		return
	}
	m = make([]string, 0)
	// SELECT FROM
	selectStmt, ok := stmt.(*influxql.SelectStatement)
	if ok {
		for _, source := range selectStmt.Sources {
			measurement, ok := source.(*influxql.Measurement)
			if ok {
				m = append(m, measurement.Name)
			}
		}
		return
	}
	// SHOW FIELD KEYS FROM
	showFieldKeyStmt, ok := stmt.(*influxql.ShowFieldKeysStatement)
	if ok {
		for _, source := range showFieldKeyStmt.Sources {
			measurement, ok := source.(*influxql.Measurement)
			if ok {
				m = append(m, measurement.Name)
			}
		}
		return
	}
	// SHOW SERIES FROM
	showSeriesStmt, ok := stmt.(*influxql.ShowSeriesStatement)
	if ok {
		for _, source := range showSeriesStmt.Sources {
			measurement, ok := source.(*influxql.Measurement)
			if ok {
				m = append(m, measurement.Name)
			}
		}
		return
	}
	// SHOW TAG KEYS FROM
	showTagKeyStmt, ok := stmt.(*influxql.ShowTagKeysStatement)
	if ok {
		for _, source := range showTagKeyStmt.Sources {
			measurement, ok := source.(*influxql.Measurement)
			if ok {
				m = append(m, measurement.Name)
			}
		}
		return
	}
	// SHOW TAG VALUES FROM
	showTagValueStmt, ok := stmt.(*influxql.ShowTagValuesStatement)
	if ok {
		for _, source := range showTagValueStmt.Sources {
			measurement, ok := source.(*influxql.Measurement)
			if ok {
				m = append(m, measurement.Name)
			}
		}
		return
	}
	// DELETE FROM
	deleteStmt, ok := stmt.(*influxql.DeleteStatement)
	if ok {
		source := deleteStmt.Source
		measurement, ok := source.(*influxql.Measurement)
		if ok {
			m = append(m, measurement.Name)
		}
		return
	}
	deleteSeriesStmt, ok := stmt.(*influxql.DeleteSeriesStatement)
	if ok {
		for _, source := range deleteSeriesStmt.Sources {
			measurement, ok := source.(*influxql.Measurement)
			if ok {
				m = append(m, measurement.Name)
			}
		}
		return
	}

	return m, ErrIllegalQL
}

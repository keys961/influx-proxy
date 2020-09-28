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

func GetMeasurementFromInfluxQL(q string) (m []string, err error) {
	stmt, err := influxql.ParseStatement(q)
	if err != nil {
		return
	}
	m = make([]string, 0)
	// SELECT
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
	// SHOW FIELD KEY
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
	// SHOW SERIES
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
	// SHOW TAG KEY
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
	// SHOW TAG VALUE
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
	return m, ErrIllegalQL
}

package mcp

import (
	"context"
	"fmt"

	"github.com/ovn-kubernetes/libovsdb/client"
	"github.com/ovn-kubernetes/libovsdb/model"
	"github.com/ovn-kubernetes/libovsdb/ovsdb"
)

// ExecuteSelectQuery is a helper function for executing select operations
func ExecuteSelectQuery[T any](ctx context.Context, client client.Client, model T, conditions ...model.Condition) ([]T, error) {
	var selectOps []ovsdb.Operation
	var queryID string
	var selectErr error

	if len(conditions) > 0 {
		selectOps, queryID, selectErr = client.WhereAll(&model, conditions...).Select()
	} else {
		selectOps, queryID, selectErr = client.Where(&model).Select()
	}

	if selectErr != nil {
		return nil, fmt.Errorf("failed to create select operation: %w", selectErr)
	}

	// Execute the transaction
	reply, err := client.Transact(ctx, selectOps...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute transaction: %w", err)
	}

	// Create a slice to hold results
	var results []T
	err = client.GetSelectResults(selectOps, reply, map[string]interface{}{queryID: &results})
	if err != nil {
		return nil, fmt.Errorf("failed to get select results: %w", err)
	}

	return results, nil
}

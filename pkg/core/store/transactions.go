/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package store

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

type txCtx struct{}

func CtxWithTx(ctx context.Context, tx Transaction) context.Context {
	return context.WithValue(ctx, txCtx{}, tx)
}

func TxFromCtx(ctx context.Context) (Transaction, bool) {
	if value, ok := ctx.Value(txCtx{}).(Transaction); ok {
		return value, true
	}
	return nil, false
}

type Transaction interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// Transactions is a producer of transactions if a resource store support transactions.
// Transactions are never required for consistency in dubbo, because there are ResourceStores that does not support transactions.
// However, in a couple of cases executing queries in transaction can improve the performance.
//
// In case of Postgres, you may set hooks when retrieve and release connections for the connection pool.
// In this case, if you create multiple resources, you need to acquire connection and execute hooks for each create.
// If you create transaction for it, you execute hooks once and you hold the connection.
//
// Transaction is propagated implicitly through Context.
type Transactions interface {
	Begin(ctx context.Context) (Transaction, error)
}

func InTx(ctx context.Context, transactions Transactions, fn func(ctx context.Context) error) error {
	tx, err := transactions.Begin(ctx)
	if err != nil {
		return err
	}
	if err := fn(CtxWithTx(ctx, tx)); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return multierr.Append(errors.Wrap(rollbackErr, "could not rollback transaction"), err)
		}
		return err
	}
	return tx.Commit(ctx)
}

type NoopTransaction struct{}

func (n NoopTransaction) Commit(context.Context) error {
	return nil
}

func (n NoopTransaction) Rollback(context.Context) error {
	return nil
}

var _ Transaction = &NoopTransaction{}

type NoTransactions struct{}

func (n NoTransactions) Begin(context.Context) (Transaction, error) {
	return NoopTransaction{}, nil
}

var _ Transactions = &NoTransactions{}

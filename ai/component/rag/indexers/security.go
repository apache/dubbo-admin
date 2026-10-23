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

package indexers

import (
	"fmt"

	"dubbo-admin-ai/security"

	"github.com/cloudwego/eino/schema"
)

var derivedDataPolicy = security.NewKnowledgeIndexPolicy()

func prepareDocuments(docs []*schema.Document) ([]*schema.Document, error) {
	prepared := make([]*schema.Document, 0, len(docs))
	for index, doc := range docs {
		if doc == nil {
			return nil, fmt.Errorf("index document is nil")
		}
		idFindings, idErr := derivedDataPolicy.ValidateIdentifier(doc.ID)
		if idErr != nil {
			return nil, fmt.Errorf("index document at position %d rejected by rule %s: %w", index, ruleID(idFindings), idErr)
		}
		metadata, findings, err := derivedDataPolicy.Prepare(doc.Content, doc.MetaData)
		if err != nil {
			ruleID := "unknown"
			if len(findings) > 0 {
				ruleID = findings[0].RuleID
			}
			return nil, fmt.Errorf("index document at position %d rejected by rule %s: %w", index, ruleID, err)
		}
		prepared = append(prepared, &schema.Document{ID: doc.ID, Content: doc.Content, MetaData: metadata})
	}
	return prepared, nil
}

func ruleID(findings []security.Finding) string {
	if len(findings) == 0 {
		return "unknown"
	}
	return findings[0].RuleID
}

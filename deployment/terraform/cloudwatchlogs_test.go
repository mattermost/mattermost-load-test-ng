package terraform

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifyPolicy(t *testing.T) {
	newMinDoc := func() policyDocument {
		return policyDocument{
			Version: "2012-10-17",
			Statement: []policyDocumentStmt{
				{
					Effect: "Allow",
					Principal: policyDocumentStmtPrincipal{
						Service: "es.amazonaws.com",
					},
					Action: []string{
						logsPermissionPutLogEventsBatch,
						logsPermissionPutLogEvents,
						logsPermissionCreateLogStream,
					},
					Resource: "arn:aws:logs:*",
				},
			},
		}
	}

	t.Run("minimum document is valid", func(t *testing.T) {
		doc := newMinDoc()
		require.True(t, verifyPolicy(doc))
	})

	t.Run("minimum document with more permission is still valid", func(t *testing.T) {
		doc := newMinDoc()
		doc.Statement[0].Action = append(doc.Statement[0].Action, "another permission")
		require.True(t, verifyPolicy(doc))
	})

	t.Run("minimum document with more statements is still valid", func(t *testing.T) {
		doc := newMinDoc()
		doc.Statement = append(doc.Statement, policyDocumentStmt{
			Effect:    "Allow",
			Principal: policyDocumentStmtPrincipal{"otherservice"},
			Action:    []string{"some other permission"},
			Resource:  "another resource",
		})
		require.True(t, verifyPolicy(doc))
	})

	t.Run("document witout permission EventsBatch makes the document invalid", func(t *testing.T) {
		doc := newMinDoc()
		doc.Statement[0].Action = slices.Delete(doc.Statement[0].Action, 0, 1)
		require.False(t, verifyPolicy(doc))
	})

	t.Run("document witout permission Events makes the document invalid", func(t *testing.T) {
		doc := newMinDoc()
		doc.Statement[0].Action = slices.Delete(doc.Statement[0].Action, 1, 2)
		require.False(t, verifyPolicy(doc))
	})

	t.Run("document witout permission Stream makes the document invalid", func(t *testing.T) {
		doc := newMinDoc()
		doc.Statement[0].Action = slices.Delete(doc.Statement[0].Action, 2, 3)
		require.False(t, verifyPolicy(doc))
	})
}

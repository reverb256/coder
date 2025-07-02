//go:build integration

package coderdtest

import (
	"context"
	"testing"
	"time"

	"github.com/coder/coder/v2/agentic"
	"github.com/coder/coder/v2/coderd/coderdtest"
	"github.com/stretchr/testify/require"
)

func TestOpenCodeAgentZero_Integration(t *testing.T) {
	t.Parallel()
	client := coderdtest.New(t, &coderdtest.Options{
		IncludeProvisionerDaemon: true,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 1. Deploy OpenCode and Agent-Zero agents via API
	// 2. Validate agent registration and health
	// 3. Create workspace using opencode-agentzero template
	// 4. Validate workspace deployment and agent communication
	// 5. Test cross-component data flow and error handling

	// TODO: Implement full integration logic with mocks and real endpoints.
	require.NotNil(t, client)
}

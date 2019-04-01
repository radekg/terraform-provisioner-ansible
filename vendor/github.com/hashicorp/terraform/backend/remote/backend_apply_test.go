package remote

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func testOperationApply() *backend.Operation {
	return &backend.Operation{
		Parallelism: defaultParallelism,
		PlanRefresh: true,
		Type:        backend.OperationTypeApply,
	}
}

func TestRemote_applyBasic(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op := testOperationApply()
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("missing apply summery in output: %s", output)
	}
}

func TestRemote_applyCanceled(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	op := testOperationApply()
	op.Module = mod
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	// Stop the run to simulate a Ctrl-C.
	run.Stop()

	<-run.Done()
	if run.ExitCode == 0 {
		t.Fatal("expected apply operation to fail")
	}
}

func TestRemote_applyWithoutPermissions(t *testing.T) {
	b, bCleanup := testBackendNoDefault(t)
	defer bCleanup()

	// Create a named workspace without permissions.
	w, err := b.client.Workspaces.Create(
		context.Background(),
		b.organization,
		tfe.WorkspaceCreateOptions{
			Name: tfe.String(b.prefix + "prod"),
		},
	)
	if err != nil {
		t.Fatalf("error creating named workspace: %v", err)
	}
	w.Permissions.CanQueueApply = false

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	op := testOperationApply()
	op.Module = mod
	op.Workspace = "prod"

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected an apply error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "insufficient rights to apply changes") {
		t.Fatalf("expected a permissions error, got: %v", run.Err)
	}
}

func TestRemote_applyWithVCS(t *testing.T) {
	b, bCleanup := testBackendNoDefault(t)
	defer bCleanup()

	// Create a named workspace with a VCS.
	_, err := b.client.Workspaces.Create(
		context.Background(),
		b.organization,
		tfe.WorkspaceCreateOptions{
			Name:    tfe.String(b.prefix + "prod"),
			VCSRepo: &tfe.VCSRepoOptions{},
		},
	)
	if err != nil {
		t.Fatalf("error creating named workspace: %v", err)
	}

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	op := testOperationApply()
	op.Module = mod
	op.Workspace = "prod"

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected an apply error, got: %v", run.Err)
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}
	if !strings.Contains(run.Err.Error(), "not allowed for workspaces with a VCS") {
		t.Fatalf("expected a VCS error, got: %v", run.Err)
	}
}

func TestRemote_applyWithParallelism(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	op := testOperationApply()
	op.Module = mod
	op.Parallelism = 3
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected an apply error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "parallelism values are currently not supported") {
		t.Fatalf("expected a parallelism error, got: %v", run.Err)
	}
}

func TestRemote_applyWithPlan(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	op := testOperationApply()
	op.Module = mod
	op.Plan = &terraform.Plan{}
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected an apply error, got: %v", run.Err)
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}
	if !strings.Contains(run.Err.Error(), "saved plan is currently not supported") {
		t.Fatalf("expected a saved plan error, got: %v", run.Err)
	}
}

func TestRemote_applyWithoutRefresh(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	op := testOperationApply()
	op.Module = mod
	op.PlanRefresh = false
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected an apply error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "refresh is currently not supported") {
		t.Fatalf("expected a refresh error, got: %v", run.Err)
	}
}

func TestRemote_applyWithTarget(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	op := testOperationApply()
	op.Module = mod
	op.Targets = []string{"null_resource.foo"}
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected an apply error, got: %v", run.Err)
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}
	if !strings.Contains(run.Err.Error(), "targeting is currently not supported") {
		t.Fatalf("expected a targeting error, got: %v", run.Err)
	}
}

func TestRemote_applyWithVariables(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply-variables")
	defer modCleanup()

	op := testOperationApply()
	op.Module = mod
	op.Variables = map[string]interface{}{"foo": "bar"}
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected an apply error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "variables are currently not supported") {
		t.Fatalf("expected a variables error, got: %v", run.Err)
	}
}

func TestRemote_applyNoConfig(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	op := testOperationApply()
	op.Module = nil
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected an apply error, got: %v", run.Err)
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}
	if !strings.Contains(run.Err.Error(), "configuration files found") {
		t.Fatalf("expected configuration files error, got: %v", run.Err)
	}
}

func TestRemote_applyNoChanges(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply-no-changes")
	defer modCleanup()

	op := testOperationApply()
	op.Module = mod
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "No changes. Infrastructure is up-to-date.") {
		t.Fatalf("expected no changes in plan summery: %s", output)
	}
}

func TestRemote_applyNoApprove(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"approve": "no",
	})

	op := testOperationApply()
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected an apply error, got: %v", run.Err)
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}
	if !strings.Contains(run.Err.Error(), "Apply discarded") {
		t.Fatalf("expected an apply discarded error, got: %v", run.Err)
	}
	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}
}

func TestRemote_applyAutoApprove(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"approve": "no",
	})

	op := testOperationApply()
	op.AutoApprove = true
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	if len(input.answers) != 1 {
		t.Fatalf("expected an unused answer, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("missing apply summery in output: %s", output)
	}
}

func TestRemote_applyApprovedExternally(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"approve": "wait-for-external-update",
	})

	op := testOperationApply()
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	ctx := context.Background()

	run, err := b.Operation(ctx, op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	// Wait 2 seconds to make sure the run started.
	time.Sleep(2 * time.Second)

	wl, err := b.client.Workspaces.List(
		ctx,
		b.organization,
		tfe.WorkspaceListOptions{
			ListOptions: tfe.ListOptions{PageNumber: 2, PageSize: 10},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error listing workspaces: %v", err)
	}
	if len(wl.Items) != 1 {
		t.Fatalf("expected 1 workspace, got %d workspaces", len(wl.Items))
	}

	rl, err := b.client.Runs.List(ctx, wl.Items[0].ID, tfe.RunListOptions{})
	if err != nil {
		t.Fatalf("unexpected error listing runs: %v", err)
	}
	if len(rl.Items) != 1 {
		t.Fatalf("expected 1 run, got %d runs", len(rl.Items))
	}

	err = b.client.Runs.Apply(context.Background(), rl.Items[0].ID, tfe.RunApplyOptions{})
	if err != nil {
		t.Fatalf("unexpected error approving run: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("missing remote backend header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "approved using the UI or API") {
		t.Fatalf("missing external approval in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("missing apply summery in output: %s", output)
	}
}

func TestRemote_applyDiscardedExternally(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"approve": "wait-for-external-update",
	})

	op := testOperationApply()
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	ctx := context.Background()

	run, err := b.Operation(ctx, op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	// Wait 2 seconds to make sure the run started.
	time.Sleep(2 * time.Second)

	wl, err := b.client.Workspaces.List(
		ctx,
		b.organization,
		tfe.WorkspaceListOptions{
			ListOptions: tfe.ListOptions{PageNumber: 2, PageSize: 10},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error listing workspaces: %v", err)
	}
	if len(wl.Items) != 1 {
		t.Fatalf("expected 1 workspace, got %d workspaces", len(wl.Items))
	}

	rl, err := b.client.Runs.List(ctx, wl.Items[0].ID, tfe.RunListOptions{})
	if err != nil {
		t.Fatalf("unexpected error listing runs: %v", err)
	}
	if len(rl.Items) != 1 {
		t.Fatalf("expected 1 run, got %d runs", len(rl.Items))
	}

	err = b.client.Runs.Discard(context.Background(), rl.Items[0].ID, tfe.RunDiscardOptions{})
	if err != nil {
		t.Fatalf("unexpected error discarding run: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected an apply error, got: %v", run.Err)
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("expected remote backend header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summery in output: %s", output)
	}
	if !strings.Contains(output, "discarded using the UI or API") {
		t.Fatalf("expected external discard output: %s", output)
	}
	if strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("unexpected apply summery in output: %s", output)
	}
}

func TestRemote_applyWithAutoApply(t *testing.T) {
	b, bCleanup := testBackendNoDefault(t)
	defer bCleanup()

	// Create a named workspace that auto applies.
	_, err := b.client.Workspaces.Create(
		context.Background(),
		b.organization,
		tfe.WorkspaceCreateOptions{
			AutoApply: tfe.Bool(true),
			Name:      tfe.String(b.prefix + "prod"),
		},
	)
	if err != nil {
		t.Fatalf("error creating named workspace: %v", err)
	}

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op := testOperationApply()
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = "prod"

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	if len(input.answers) != 1 {
		t.Fatalf("expected an unused answer, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("missing apply summery in output: %s", output)
	}
}

func TestRemote_applyForceLocal(t *testing.T) {
	// Set TF_FORCE_LOCAL_BACKEND so the remote backend will use
	// the local backend with itself as embedded backend.
	if err := os.Setenv("TF_FORCE_LOCAL_BACKEND", "1"); err != nil {
		t.Fatalf("error setting environment variable TF_FORCE_LOCAL_BACKEND: %v", err)
	}
	defer os.Unsetenv("TF_FORCE_LOCAL_BACKEND")

	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op := testOperationApply()
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("unexpected remote backend header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summery in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("expected apply summery in output: %s", output)
	}
}

func TestRemote_applyWorkspaceWithoutOperations(t *testing.T) {
	b, bCleanup := testBackendNoDefault(t)
	defer bCleanup()

	ctx := context.Background()

	// Create a named workspace that doesn't allow operations.
	_, err := b.client.Workspaces.Create(
		ctx,
		b.organization,
		tfe.WorkspaceCreateOptions{
			Name: tfe.String(b.prefix + "no-operations"),
		},
	)
	if err != nil {
		t.Fatalf("error creating named workspace: %v", err)
	}

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op := testOperationApply()
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = "no-operations"

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if strings.Contains(output, "Running apply in the remote backend") {
		t.Fatalf("unexpected remote backend header in output: %s", output)
	}
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("expected plan summery in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("expected apply summery in output: %s", output)
	}
}

func TestRemote_applyLockTimeout(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	ctx := context.Background()

	// Retrieve the workspace used to run this operation in.
	w, err := b.client.Workspaces.Read(ctx, b.organization, b.workspace)
	if err != nil {
		t.Fatalf("error retrieving workspace: %v", err)
	}

	// Create a new configuration version.
	c, err := b.client.ConfigurationVersions.Create(ctx, w.ID, tfe.ConfigurationVersionCreateOptions{})
	if err != nil {
		t.Fatalf("error creating configuration version: %v", err)
	}

	// Create a pending run to block this run.
	_, err = b.client.Runs.Create(ctx, tfe.RunCreateOptions{
		ConfigurationVersion: c,
		Workspace:            w,
	})
	if err != nil {
		t.Fatalf("error creating pending run: %v", err)
	}

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"cancel":  "yes",
		"approve": "yes",
	})

	op := testOperationApply()
	op.StateLockTimeout = 5 * time.Second
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	_, err = b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, syscall.SIGINT)
	select {
	case <-sigint:
		// Stop redirecting SIGINT signals.
		signal.Stop(sigint)
	case <-time.After(10 * time.Second):
		t.Fatalf("expected lock timeout after 5 seconds, waited 10 seconds")
	}

	if len(input.answers) != 2 {
		t.Fatalf("expected unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Lock timeout exceeded") {
		t.Fatalf("missing lock timout error in output: %s", output)
	}
	if strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("unexpected plan summery in output: %s", output)
	}
	if strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("unexpected apply summery in output: %s", output)
	}
}

func TestRemote_applyDestroy(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply-destroy")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op := testOperationApply()
	op.Destroy = true
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "0 to add, 0 to change, 1 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "0 added, 0 changed, 1 destroyed") {
		t.Fatalf("missing apply summery in output: %s", output)
	}
}

func TestRemote_applyDestroyNoConfig(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op := testOperationApply()
	op.Destroy = true
	op.Module = nil
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("unexpected apply error: %v", run.Err)
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}
}

func TestRemote_applyPolicyPass(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply-policy-passed")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op := testOperationApply()
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: true") {
		t.Fatalf("missing polic check result in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("missing apply summery in output: %s", output)
	}
}

func TestRemote_applyPolicyHardFail(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply-policy-hard-failed")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"approve": "yes",
	})

	op := testOperationApply()
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected an apply error, got: %v", run.Err)
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}
	if !strings.Contains(run.Err.Error(), "hard failed") {
		t.Fatalf("expected a policy check error, got: %v", run.Err)
	}
	if len(input.answers) != 1 {
		t.Fatalf("expected an unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: false") {
		t.Fatalf("missing policy check result in output: %s", output)
	}
	if strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("unexpected apply summery in output: %s", output)
	}
}

func TestRemote_applyPolicySoftFail(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply-policy-soft-failed")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"override": "override",
		"approve":  "yes",
	})

	op := testOperationApply()
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	if len(input.answers) > 0 {
		t.Fatalf("expected no unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: false") {
		t.Fatalf("missing policy check result in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("missing apply summery in output: %s", output)
	}
}

func TestRemote_applyPolicySoftFailAutoApprove(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply-policy-soft-failed")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"override": "override",
	})

	op := testOperationApply()
	op.AutoApprove = true
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected an apply error, got: %v", run.Err)
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}
	if !strings.Contains(run.Err.Error(), "soft failed") {
		t.Fatalf("expected a policy check error, got: %v", run.Err)
	}
	if len(input.answers) != 1 {
		t.Fatalf("expected an unused answers, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: false") {
		t.Fatalf("missing policy check result in output: %s", output)
	}
	if strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("unexpected apply summery in output: %s", output)
	}
}

func TestRemote_applyPolicySoftFailAutoApply(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	// Create a named workspace that auto applies.
	_, err := b.client.Workspaces.Create(
		context.Background(),
		b.organization,
		tfe.WorkspaceCreateOptions{
			AutoApply: tfe.Bool(true),
			Name:      tfe.String(b.prefix + "prod"),
		},
	)
	if err != nil {
		t.Fatalf("error creating named workspace: %v", err)
	}

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply-policy-soft-failed")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"override": "override",
		"approve":  "yes",
	})

	op := testOperationApply()
	op.Module = mod
	op.UIIn = input
	op.UIOut = b.CLI
	op.Workspace = "prod"

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	if len(input.answers) != 1 {
		t.Fatalf("expected an unused answer, got: %v", input.answers)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: false") {
		t.Fatalf("missing policy check result in output: %s", output)
	}
	if !strings.Contains(output, "1 added, 0 changed, 0 destroyed") {
		t.Fatalf("missing apply summery in output: %s", output)
	}
}

func TestRemote_applyWithRemoteError(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	mod, modCleanup := module.TestTree(t, "./test-fixtures/apply-with-error")
	defer modCleanup()

	op := testOperationApply()
	op.Module = mod
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("error running operation: %v", run.Err)
	}
	if run.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", run.ExitCode)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "null_resource.foo: 1 error") {
		t.Fatalf("missing apply error in output: %s", output)
	}
}

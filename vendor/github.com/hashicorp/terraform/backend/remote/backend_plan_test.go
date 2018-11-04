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

func testOperationPlan() *backend.Operation {
	return &backend.Operation{
		ModuleDepth: defaultModuleDepth,
		Parallelism: defaultParallelism,
		PlanRefresh: true,
		Type:        backend.OperationTypePlan,
	}
}

func TestRemote_planBasic(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	op := testOperationPlan()
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
	if run.PlanEmpty {
		t.Fatal("expected a non-empty plan")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
}

func TestRemote_planWithoutPermissions(t *testing.T) {
	b := testBackendNoDefault(t)

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
	w.Permissions.CanQueueRun = false

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	op := testOperationPlan()
	op.Module = mod
	op.Workspace = "prod"

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected a plan error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "insufficient rights to generate a plan") {
		t.Fatalf("expected a permissions error, got: %v", run.Err)
	}
}

func TestRemote_planWithModuleDepth(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	op := testOperationPlan()
	op.Module = mod
	op.ModuleDepth = 1
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected a plan error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "module depths are currently not supported") {
		t.Fatalf("expected a module depth error, got: %v", run.Err)
	}
}

func TestRemote_planWithParallelism(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	op := testOperationPlan()
	op.Module = mod
	op.Parallelism = 3
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected a plan error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "parallelism values are currently not supported") {
		t.Fatalf("expected a parallelism error, got: %v", run.Err)
	}
}

func TestRemote_planWithPlan(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	op := testOperationPlan()
	op.Module = mod
	op.Plan = &terraform.Plan{}
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected a plan error, got: %v", run.Err)
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}
	if !strings.Contains(run.Err.Error(), "saved plan is currently not supported") {
		t.Fatalf("expected a saved plan error, got: %v", run.Err)
	}
}

func TestRemote_planWithPath(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	op := testOperationPlan()
	op.Module = mod
	op.PlanOutPath = "./test-fixtures/plan"
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected a plan error, got: %v", run.Err)
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}
	if !strings.Contains(run.Err.Error(), "generated plan is currently not supported") {
		t.Fatalf("expected a generated plan error, got: %v", run.Err)
	}
}

func TestRemote_planWithoutRefresh(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	op := testOperationPlan()
	op.Module = mod
	op.PlanRefresh = false
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected a plan error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "refresh is currently not supported") {
		t.Fatalf("expected a refresh error, got: %v", run.Err)
	}
}

func TestRemote_planWithTarget(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	op := testOperationPlan()
	op.Module = mod
	op.Targets = []string{"null_resource.foo"}
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected a plan error, got: %v", run.Err)
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}
	if !strings.Contains(run.Err.Error(), "targeting is currently not supported") {
		t.Fatalf("expected a targeting error, got: %v", run.Err)
	}
}

func TestRemote_planWithVariables(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	op := testOperationPlan()
	op.Module = mod
	op.Variables = map[string]interface{}{"foo": "bar"}
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected an plan error, got: %v", run.Err)
	}
	if !strings.Contains(run.Err.Error(), "variables are currently not supported") {
		t.Fatalf("expected a variables error, got: %v", run.Err)
	}
}

func TestRemote_planNoConfig(t *testing.T) {
	b := testBackendDefault(t)

	op := testOperationPlan()
	op.Module = nil
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err == nil {
		t.Fatalf("expected a plan error, got: %v", run.Err)
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}
	if !strings.Contains(run.Err.Error(), "configuration files found") {
		t.Fatalf("expected configuration files error, got: %v", run.Err)
	}
}

func TestRemote_planLockTimeout(t *testing.T) {
	b := testBackendDefault(t)
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

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	input := testInput(t, map[string]string{
		"cancel":  "yes",
		"approve": "yes",
	})

	op := testOperationPlan()
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
}

func TestRemote_planDestroy(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan")
	defer modCleanup()

	op := testOperationPlan()
	op.Destroy = true
	op.Module = mod
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("unexpected plan error: %v", run.Err)
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}
}

func TestRemote_planDestroyNoConfig(t *testing.T) {
	b := testBackendDefault(t)

	op := testOperationPlan()
	op.Destroy = true
	op.Module = nil
	op.Workspace = backend.DefaultStateName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Err != nil {
		t.Fatalf("unexpected plan error: %v", run.Err)
	}
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}
}

func TestRemote_planWithWorkingDirectory(t *testing.T) {
	b := testBackendDefault(t)

	options := tfe.WorkspaceUpdateOptions{
		WorkingDirectory: tfe.String("terraform"),
	}

	// Configure the workspace to use a custom working direcrtory.
	_, err := b.client.Workspaces.Update(context.Background(), b.organization, b.workspace, options)
	if err != nil {
		t.Fatalf("error configuring working directory: %v", err)
	}

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan-with-working-directory/terraform")
	defer modCleanup()

	op := testOperationPlan()
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
	if run.PlanEmpty {
		t.Fatalf("expected a non-empty plan")
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
}

func TestRemote_planPolicyPass(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan-policy-passed")
	defer modCleanup()

	input := testInput(t, map[string]string{})

	op := testOperationPlan()
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

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: true") {
		t.Fatalf("missing polic check result in output: %s", output)
	}
}

func TestRemote_planPolicyHardFail(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan-policy-hard-failed")
	defer modCleanup()

	input := testInput(t, map[string]string{})

	op := testOperationPlan()
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
		t.Fatalf("expected a plan error, got: %v", run.Err)
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}
	if !strings.Contains(run.Err.Error(), "hard failed") {
		t.Fatalf("expected a policy check error, got: %v", run.Err)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: false") {
		t.Fatalf("missing policy check result in output: %s", output)
	}
}

func TestRemote_planPolicySoftFail(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan-policy-soft-failed")
	defer modCleanup()

	input := testInput(t, map[string]string{})

	op := testOperationPlan()
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
		t.Fatalf("expected a plan error, got: %v", run.Err)
	}
	if !run.PlanEmpty {
		t.Fatalf("expected plan to be empty")
	}
	if !strings.Contains(run.Err.Error(), "soft failed") {
		t.Fatalf("expected a policy check error, got: %v", run.Err)
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "1 to add, 0 to change, 0 to destroy") {
		t.Fatalf("missing plan summery in output: %s", output)
	}
	if !strings.Contains(output, "Sentinel Result: false") {
		t.Fatalf("missing policy check result in output: %s", output)
	}
}

func TestRemote_planWithRemoteError(t *testing.T) {
	b := testBackendDefault(t)

	mod, modCleanup := module.TestTree(t, "./test-fixtures/plan-with-error")
	defer modCleanup()

	op := testOperationPlan()
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
		t.Fatalf("missing plan error in output: %s", output)
	}
}

package console

import (
	"testing"
)

// recordingCommand captures whether Handle ran and the parameters it validated.
type recordingCommand struct {
	Command
	handled bool
	params  CommandParameter
}

func (c *recordingCommand) Validate(parameters CommandParameter) error {
	c.params = parameters
	return nil
}

func (c *recordingCommand) Handle() {
	c.handled = true
}

func TestRunCommandsParsesParameters(t *testing.T) {
	cmd := &recordingCommand{}
	RegisterCommand(cmd, "greet")

	RunCommands([]string{
		"greet",
		"--name=John",
		"--email=John Land<john@mail.com>",
		"--married=true",
	})

	if !cmd.handled {
		t.Fatal("expected command Handle to run")
	}
	if got := cmd.params["name"]; got != "John" {
		t.Errorf("name = %q, want %q", got, "John")
	}
	if got := cmd.params["email"]; got != "John Land<john@mail.com>" {
		t.Errorf("email = %q, want %q", got, "John Land<john@mail.com>")
	}
	if got := cmd.params["married"]; got != "true" {
		t.Errorf("married = %q, want %q", got, "true")
	}
}

func TestRunCommandsEmptyArgsDoesNotPanic(t *testing.T) {
	// Must not panic on an empty argument slice.
	RunCommands(nil)
	RunCommands([]string{})
}

func TestRunCommandsMalformedParameterIsSkipped(t *testing.T) {
	cmd := &recordingCommand{}
	RegisterCommand(cmd, "skip")

	// "--broken" has no `=value`; it must be ignored rather than panic.
	RunCommands([]string{"skip", "--broken", "--valid=yes"})

	if !cmd.handled {
		t.Fatal("expected command Handle to run")
	}
	if _, ok := cmd.params["broken"]; ok {
		t.Error("malformed parameter should not be recorded")
	}
	if got := cmd.params["valid"]; got != "yes" {
		t.Errorf("valid = %q, want %q", got, "yes")
	}
}

func TestRunCommandsUnknownCommandIsNoop(t *testing.T) {
	// Unknown command name should be handled gracefully.
	RunCommands([]string{"does-not-exist"})
}

package console

import (
	"github.com/gflydev/core/log"
	"github.com/gflydev/core/utils"
	"regexp"
)

// ===========================================================================================================
// 												Command
// ===========================================================================================================

// ICommand The interface command.
type ICommand interface {
	Validate(parameters CommandParameter) error
	Handle()
}

// Command Abstract command.
type Command struct{}

func (c Command) Validate(parameters CommandParameter) error {
	return nil
}

func (c Command) Handle() {
	log.Panic("Must be implemented from children")
}

// ===========================================================================================================
// 											Command Handler
// ===========================================================================================================

// CommandParameter a map to store parameters
type CommandParameter map[string]interface{}

// Command pool
var commands = make(map[string]ICommand)

// paramPattern matches a single `--key=value` argument.
// Compiled once at package load instead of on every argument.
// Parameter pattern `--age=41 --name=John --email="John Land<john@mail.com>" --married=true`
var paramPattern = regexp.MustCompile(`--(\w+)=([\w*\s+"'<>@./#&-]*)`)

// RegisterCommand add a new command to pool.
func RegisterCommand(cmd ICommand, name ...string) {
	cmdName := utils.ReflectType(cmd)[1:]

	if len(name) > 0 {
		cmdName = name[0]
	}

	commands[cmdName] = cmd
}

func RunCommands(args []string) {
	if len(args) == 0 {
		log.Info("No command name provided")
		return
	}

	cmdName := args[0]
	cmd, ok := commands[cmdName]

	if !ok {
		log.Infof("Don't see command name `%s`", cmdName)
		return
	}

	// Parse `--key=value` arguments, skipping anything that doesn't match.
	var parameters = CommandParameter{}
	for _, param := range args[1:] {
		matches := paramPattern.FindStringSubmatch(param)
		if matches == nil {
			log.Infof("Ignoring malformed parameter `%s`", param)
			continue
		}

		parameters[matches[1]] = matches[2]
	}

	err := cmd.Validate(parameters)
	if err != nil {
		log.Errorf("Error: %v", err)

		return
	}

	cmd.Handle()
}

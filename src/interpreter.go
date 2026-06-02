package main

import (
	"fmt"
	"regexp"
	"strings"
)

var functions = map[string]func(*App, []string) (any, error){
	"send_message": func(app *App, args []string) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("send_message requires 2 arguments: channel and message")
		}
		recipient := args[0]
		message := args[1]

		recipient = strings.ReplaceAll(recipient, "\"", "")
		return app.Discord.ChannelMessageSend(recipient, message)
	},
}

func InterpretLine(code string) (*FunctionCall, error) {
	exp := "([a-zA-Z_]\\w+)\\(([^)]+)\\)"
	regex, err := regexp.Compile(exp)
	if err != nil {
		return nil, err
	}

	result := regex.FindStringSubmatch(code)
	if len(result) < 3 {
		return nil, fmt.Errorf("invalid function call: %s", code)
	}

	return &FunctionCall{
		Name:      result[1],
		Arguments: strings.Split(result[2], ", "),
	}, nil
}

func CallFunction(call FunctionCall, app *App) (any, error) {
	function, exists := functions[call.Name]
	if !exists {
		return nil, fmt.Errorf("function %s not found", call.Name)
	}

	result, err := function(app, call.Arguments)
	if err != nil {
		return nil, err
	}

	return result, nil
}

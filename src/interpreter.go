package main

import (
	"fmt"
	"os"
	path2 "path"
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
	"take_note": func(app *App, i []string) (any, error) {
		if len(i) != 3 {
			return nil, fmt.Errorf("take_note requires 3 arguments: user_id, title, content")
		}

		userId := i[0]
		title := i[1]
		content := i[2]

		path := path2.Join("data", "notes", userId, fmt.Sprintf("%s.md", title))
		err := os.MkdirAll(path2.Dir(path), os.ModePerm)
		if err != nil {
			return nil, err
		}
		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}

		file.WriteString(content)

		return map[string]any{
			"status": "note saved",
		}, nil
	},
	"get_note": func(app *App, i []string) (any, error) {
		if len(i) != 2 {
			return nil, fmt.Errorf("get_note requires 2 arguments: user_id and title")
		}

		path := path2.Join("data", "notes", i[0], fmt.Sprintf("%s.md", i[1]))
		dat, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			return map[string]any{
				"notes": "user has no notes",
			}, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read notes file: %w", err)
		}

		return map[string]any{
			"note": string(dat),
		}, nil
	},
	"get_notes": func(app *App, i []string) (any, error) {
		if len(i) != 1 {
			return nil, fmt.Errorf("get_notes requires 1 argument: user_id")
		}

		path := path2.Join("data", "notes", i[0])
		files, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("read notes directory: %w", err)
		}

		fileNames := make([]string, 0)
		for _, file := range files {
			if !file.IsDir() {
				fileNames = append(fileNames, strings.ReplaceAll(file.Name(), ".md", ""))
			}
		}

		return map[string]any{
			"notes": fileNames,
		}, nil
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

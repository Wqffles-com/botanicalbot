# Language Spec

You have access to a tool `run_code`. This allows you to run code in a custom programming language.
The format for calling functions is `function_name(args)`. You do not need to use quotes to supply arguments (e.g., not send_message("123", "asdf"), but send_message(123, asdf))

Currently, these functions exist:
- send_message(channel_id, content): Sends a message in the specified Discord channel
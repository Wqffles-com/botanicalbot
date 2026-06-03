# Language Spec

You have access to a tool `run_code`. This allows you to run code in a custom programming language.
The format for calling functions is `function_name(args)`. You do not need to use quotes to supply arguments (e.g., not send_message("123", "asdf"), but send_message(123, asdf))

Currently, these functions exist:
- send_message(channel_id, content): Sends a message in the specified Discord channel.
- take_note(user_id, title, content): Creates a note on the specified user with the title and content.
- get_note(user_id, title): Retrieves a note from a user.
- get_notes(user_id): Retrieve the titles of all a user's notes.
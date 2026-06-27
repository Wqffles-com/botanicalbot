# Botanical Bot

Botanical is an open source program that allows you to connect your preferred AI model directly to Discord.

Currently, the features are quite limited as I've been working on other projects, but you're welcome to contribute.

I'd recommend to contribute from the `dev` branch and not the `main` branch, as the dev branch has some pretty new features.

If you have any questions, don't hesitate to [shoot me a message on Discord](https://discord.com/users/555443167244189697).

## Usage

To use this bot you must have the newest version of Go installed. This can be downloaded at [https://go.dev](https://go.dev/).

Then, clone the repo using git: `git clone https://github.com/Wqffles-com/botanicalbot`. I recommend also switching to the `dev` branch for the newest features: `git checkout dev`.

You need to configure the bot for it to work properly. This can be done using the `data/config.toml` file. An example you can fill out can be found below.

```toml
[Discord]
Token = ""

[OpenAI]
ApiKey = ""
Model = ""
BaseUrl = ""
```

## Current Features

I strongly advise against the use of the main branch, as it lacks features. It currently has the following features:
- Chatting with the AI
- Message logging
- Tool Calls
  - Taking notes on user
  - Getting notes

The dev branch has all the features of the main branch, and:
- Sending messages in a custom specified channel (just plain text for now)

## Contributing

Please do contribute! I could really use the help as I have other projects that take up a lot of my time. I'm sorry if the code is badly documented, I didn't get to doing this.

The code should be self-explanatory in most parts, but some parts might be gibberish. You can always contact me if you want.

Please contribute from the `dev` branch and not the `main` branch, due to the migration from tool calls to a custom language. Thanks!

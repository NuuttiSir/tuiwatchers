# TuiWatchers

TuiWatchers, because it's TUI ( kinda :D ) and you can watch your favorite
Twitch streamers

## Features

- **Live stream browser** -> lists all channels you follow that are currently live
- **One-key playback** -> press Enter to start streaming in MPV
- **Built-in chat** -> spawns a dedicated terminal window with live Twitch chat
                        using WebSockets
- **Emote support** -> Chat renders Twitch emotes
- **Send messages** -> reply to chat directly from the terminal
- **OAuth login** -> Easy authentication with just two clicks
- **Session persistance** -> Succeeding re-authentications use token saved in tokens.json

## Description

<!-- TODO: ADD GIF TO SHOW APP IN ACTION! -->

TuiWatchers is a different way to watch your favorite streamers in the
terminal, well kind of. ( No way to watch videos straight in terminal, yet ;) )

TuiWatchers is simple and sometimes the better way to consume different games
and genres.

## Motivation

The idea behind TuiWatchers was my own interest on everything terminal, like
NeoVim, Yazi etc. I have always wanted to learn APIs and this seemed like the
perfect idea for it.

And I did not find any similar projects (skill issue?) so I built the thing I
wanted for myself and then it expanded to this project.

This project made me fall in love with programming all over again after kind
of burning myself out in school and useless AI-Slopping^tm around

So enjoy, please <3

## Quick Start

### 1. Check requirements

- mpv
- yt-dlp
- terminal, like ghostty
- browser, like firefox

### 2. Install TuiWatchers using Go toolchain

    go install github.com/NuuttiSir/tuiwatchers@latest

### 3. Install from github

    git clone <url>
    cd tuiwatchers
    go build

### 4. RUN THE PROGRAM

    ./tuiwatchers

## Usage

<!-- TODO: TODO -->

## Contributing

### Clone the repo

    git clone <url>
    cd tuiwatchers

### Build the compiled binary

    go build

### Submit pull request

If you would like to contribute, please fork the repository and open a pull
request to the 'main' branch. Thankz<3

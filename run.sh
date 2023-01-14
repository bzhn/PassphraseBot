if [ $(echo -n "$PASSPHRASEBOT_TOKEN" | wc -m) -lt 20 ]; then
    echo "Please specify the token for the bot: PASSPHRASEBOT_TOKEN=<BOT_TOKEN> bash run.sh"
    exit 1
fi

if ! [ -f ".env" ]; then
    echo "PASSPHRASEBOT_TOKEN=$PASSPHRASEBOT_TOKEN" > .env
    echo "Created the file with secrets"
fi

docker compose up -d

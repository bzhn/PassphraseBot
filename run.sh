if ! [ -f ".env" ]; then # if there is no .env file
    echo -n "PassphraseBot Token: "
    read PASSPHRASEBOT_TOKEN

    if [ $(echo -n "$PASSPHRASEBOT_TOKEN" | wc -m) -lt 20 ]; then
        echo "Please specify the token for the bot: PASSPHRASEBOT_TOKEN=<BOT_TOKEN> bash run.sh"
        exit 1
    fi

    echo "PASSPHRASEBOT_TOKEN=$PASSPHRASEBOT_TOKEN" > .env
    echo "Created the file with secrets"
fi

docker compose up --build -d

rm .env
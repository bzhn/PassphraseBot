# PassphraseBot

This is a Telegram Bot which helps you generate mnemonic passphrases and store them in the database.

## Overview

On `/start` command bot sends a greeting message a and simple instruction.
Also **Generate** button appears at the bottom of chat:
![image](https://user-images.githubusercontent.com/89320434/202588416-a9c6c373-393d-45bf-a29a-3c1154c04c69.png)

After user clicks **Generate**, bot deletes his message and sends a new passphrase:
![image](https://user-images.githubusercontent.com/89320434/202588491-56b9371c-248a-44b9-981d-0a4e58a429bf.png)

If user wants to generate another passphrase within the same message, he can click **Regenerate passphrase** inline button:
![image](https://user-images.githubusercontent.com/89320434/202588868-f7914579-7195-4271-a561-5970de02cbc8.png)

Here are the commands that are waiting to be implemented
![image](https://user-images.githubusercontent.com/89320434/202588915-4f7c8c7b-6116-4226-9f52-e660e50f35c9.png)

## Roadmap

- [x] Generate passphrase
- [x] Delete and regenerate existing passphrase
- [ ] Choose custom wordlist
- [ ] Add encryption password to the account
- [ ] Encrypt and store passphrases in the database
- [ ] Save passphrases with custom notes
- [ ] List saved passphrases
- [ ] Search through passphrase notes

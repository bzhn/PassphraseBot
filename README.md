<div align="center">
  <img src="./assets/logo.jpg" width="100px" />
  <h1 id="erw">PassphraseBot</h1>
  <h3>Telegram Bot for generating mnemonic Passphrases</h3>

  <a href="https://t.me/PassphraseBot"><img src="https://img.shields.io/badge/Open in Telegram-27A7E7" alt="Telegram PassphraseBot Link" /></a>
</div>

## Overview

On `/start` command bot sends a greeting message a and simple instruction.
Also **Generate** button appears at the bottom of chat:

<div align="center">
<img width="400px" src="https://user-images.githubusercontent.com/89320434/202588416-a9c6c373-393d-45bf-a29a-3c1154c04c69.png" />
</div>

After user clicks **Generate**, bot deletes his message and sends a new passphrase:

<div align="center">
<img width="400px" src="https://user-images.githubusercontent.com/89320434/202588491-56b9371c-248a-44b9-981d-0a4e58a429bf.png" />
</div>

If user wants to generate another passphrase within the same message, he can click **Regenerate passphrase** inline button:

<div align="center">
<img width="400px" src="https://user-images.githubusercontent.com/89320434/202588868-f7914579-7195-4271-a561-5970de02cbc8.png" />
</div>

Here are the commands that are waiting to be implemented

<div align="center">
<img width="400px" src="https://user-images.githubusercontent.com/89320434/202588915-4f7c8c7b-6116-4226-9f52-e660e50f35c9.png" />
</div>

## Roadmap

- [x] Generate passphrase
- [x] Delete and regenerate existing passphrase
- [x] Choose custom wordlist
- [ ] Add encryption password to the account
- [ ] Encrypt and store passphrases in the database
- [ ] Save passphrases with custom notes
- [ ] List saved passphrases
- [ ] Search through passphrase notes

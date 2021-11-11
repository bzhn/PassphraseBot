const secret = require("./secret.json")
const TOKEN = secret.BotToken;
const TelegramBot = require('node-telegram-bot-api');
const bot = new TelegramBot(TOKEN, {polling: true});
const passph = require('passph');

bot.on('message', (msg) => {
    // Default values of amount of words and delimiter.
    let amount = 3;
    let delimiter = ' ';

    if (msg.text[0] != '/' || msg.text.length == 1) { // If it isn't a command.
        let re = /([\d]*)[ ]*([\D\S]?)$/;
        let match = re.exec(msg.text)
        if (match[1] != null && match[1] != 0 && match[1] != '' && match[1] <= 500) { // If specified not-null amount of words and it's less than 500.
            console.log(`Amount: ${match[1]}`)
            amount = match[1];
        }
        
        // Change delimiter if specified.
        if (match[2] != null && match[2] != '') {
            console.log(`Delimiter: ${match[2]}`)
            delimiter = match[2];
        }
    }
    bot.deleteMessage(msg.chat.id, msg.message_id);
    bot.sendMessage(msg.chat.id, `<code>${passph.gen(Number(amount), delimiter).replace(/</g, "&lt;") /* replace < to HTML entitie */}</code>`, {parse_mode: 'HTML'});
})

bot.on('polling_error', (error) => {
    console.error(error.code);
    console.error(error.response);
});

#!/usr/bin/env python3
# -*- coding: utf-8 -*-
import html
import logging
import sys
from configparser import ConfigParser
from datetime import datetime
from json import loads

import redis
import telegram
from requests import get
from telegram.ext import CommandHandler, Updater
from telegram.ext.dispatcher import run_async

# Logging
logging.basicConfig(
    format="%(asctime)s - %(levelname)s: %(message)s",
    datefmt="%d.%m.%Y %H:%M:%S",
    level=logging.INFO
)
logger = logging.getLogger(__name__)

# Bot configuration
config = ConfigParser()
try:
    config.read_file(open('config.ini'))
except FileNotFoundError:
    logger.critical('Config.ini nicht gefunden')
    sys.exit(1)

# Bot token
try:
    bot_token = config['DEFAULT']['token']
except KeyError:
    logger.error('Kein Bot-Token gesetzt, bitte config.ini prüfen')
    sys.exit(1)
if not bot_token:
    logger.error('Kein Bot-Token gesetzt, bitte config.ini prüfen')
    sys.exit(1)

# Admins
try:
    admins = loads(config["ADMIN"]["id"])
except KeyError:
    logger.error('Keine Admin-IDs gesetzt, bitte config.ini prüfen.')
    admins = []
if not admins:
    logger.error('Keine Admin-IDs gesetzt, bitte config.ini prüfen.')
    admins = []

for admin in admins:
    if not isinstance(admin, int):
        logger.error('Admin-IDs müssen Integer sein.')
        admins.remove(admin)

# Redis
redis_conf = config['REDIS']
redis_db = redis_conf.get('db', 0)
redis_host = redis_conf.get('host', '127.0.0.1')
redis_port = redis_conf.get('port', 6379)
redis_socket = redis_conf.get('socket_path')
if redis_socket:
    r = redis.Redis(unix_socket_path=redis_socket, db=int(redis_db), decode_responses=True)
else:
    r = redis.Redis(host=redis_host, port=int(redis_port), db=int(redis_db), decode_responses=True)

if not r.ping():
    logging.getLogger("Redis").critical("Redis-Verbindungsfehler, config.ini prüfen")
    sys.exit(1)

subscriber_hash = 'pythonbot:tagesschau:subs'
last_entry_hash = 'pythonbot:tagesschau:last_entry'


@run_async
def start(bot, update):
    if not r.sismember(subscriber_hash, update.message.chat_id):
        r.sadd(subscriber_hash, update.message.chat_id)
        logger.info('Neuer Abonnent: ' + str(update.message.chat_id))
        text = '<b>Du erhältst jetzt neue Eilmeldungen!</b>\n'
        text += 'Nutze /stop, um keine Eilmeldungen mehr zu erhalten.\n'
        text += 'Für neue Tagesschau-Artikel, check doch mal den @TagesschauDE-Kanal.\n\n'
        text += '<b>ACHTUNG:</b> Wenn du den Bot blockierst oder aus der Gruppe entfernst, '
        text += 'musst du die Eilmeldungen erneut abonnieren!'
    else:
        text = '<b>Du erhältst bereits Eilmeldungen.</b> Nutze /stop zum Deabonnieren.'
    update.message.reply_text(text, parse_mode=telegram.ParseMode.HTML)


@run_async
def stop(bot, update):
    if r.sismember(subscriber_hash, update.message.chat_id):
        r.srem(subscriber_hash, update.message.chat_id)
        logger.info('Abonnement beendet: ' + str(update.message.chat_id))
        text = '<b>Du erhältst jetzt keine Eilmeldungen mehr.</b>\n'
        text += 'Nutze /start, um wieder Eilmeldungen zu erhalten.'
    else:
        text = 'Du hast die Eilmeldungen bereits deabonniert. Mit /start kannst du diese wieder abonnieren.'
    update.message.reply_text(text, parse_mode=telegram.ParseMode.HTML)


@run_async
def help_text(bot, update):
    text = '/start: Eilmeldungen erhalten\n'
    text += '/stop: Eilmeldungen nicht mehr erhalten'
    update.message.reply_text(text)


@run_async
def run_job_manually(bot, update):
    if update.message.chat.id not in admins:
        return
    run_job(bot)


@run_async
def run_job(bot, job=None):
    logger.info('Prüfe auf neue Eilmeldungen')
    res = get('http://www.tagesschau.de/api/index.json')
    if res.status_code != 200:
        logger.warning('HTTP-Fehler ' + str(res.status_code))
        return

    try:
        data = loads(res.text)
    except ValueError:
        logger.warning('Kein valides JSON erhalten')
        return

    breakingnews = data['breakingnews']
    if not breakingnews:
        logger.debug('Keine neuen Eilmeldungen')
        return

    last_breaking = r.get(last_entry_hash)
    if not last_breaking or breakingnews[0]['date'] != last_breaking:
        logger.info('Neue Eilmeldung')
        title = html.escape(breakingnews[0]['headline'])
        if 'shorttext' not in breakingnews[0]:
            news = ''
        else:
            news = html.escape(breakingnews[0]['shorttext']).strip() + '\n'
        post_url = breakingnews[0]['details']
        post_url = post_url.replace('/api/', '/')
        post_url = post_url.replace('.json', '.html')
        posted_at = data["date"].replace("+02:00", "+0200")
        posted_at = datetime.strptime(posted_at, "%Y-%m-%dT%H:%M:%S.%f%z")
        posted_at = posted_at.strftime("%d.%m.%Y um %H:%M:%S Uhr")
        text = '<b>' + title + '</b>\n'
        text += '<i>' + posted_at + '</i>\n'
        text += news
        text_link = '<a href="' + post_url + '">Eilmeldung aufrufen</a>'
        reply_markup = telegram.InlineKeyboardMarkup(
            [
                [
                    telegram.InlineKeyboardButton(text='Eilmeldung aufrufen', url=post_url)
                ]
            ]
        )
        r.set(last_entry_hash, breakingnews[0]['date'])
        for member in r.smembers(subscriber_hash):
            try:
                if int(member) < 0:  # Group
                    bot.sendMessage(
                        chat_id=member,
                        text='#EIL: ' + text,
                        parse_mode=telegram.ParseMode.HTML,
                        disable_web_page_preview=True,
                        reply_markup=reply_markup
                    )
                else:
                    bot.sendMessage(
                        chat_id=member,
                        text=text + text_link,
                        parse_mode=telegram.ParseMode.HTML,
                        disable_web_page_preview=True
                    )
            except telegram.error.Unauthorized:
                logging.warning('Chat ' + member + ' existiert nicht mehr, wird gelöscht.')
                r.srem(subscriber_hash, member)
            except telegram.error.ChatMigrated as new_chat:
                new_chat_id = new_chat.new_chat_id
                logging.info('Chat migriert: ' + member + ' -> ' + str(new_chat_id))
                r.srem(subscriber_hash, member)
                r.sadd(subscriber_hash, new_chat_id)
                bot.sendMessage(
                    chat_id=member,
                    text=text,
                    parse_mode=telegram.ParseMode.HTML,
                    disable_web_page_preview=True,
                    reply_markup=reply_markup
                )
            except telegram.error.TimedOut:
                pass


# Main function
def main():
    # Setup the updater and show bot info
    updater = Updater(token=bot_token)
    try:
        bot_info = updater.bot.getMe()
    except telegram.error.Unauthorized:
        logger.error('Anmeldung nicht möglich, Bot-Token falsch?')
        sys.exit(1)

    logger.info('Starte ' + bot_info.first_name + ', AKA @' + bot_info.username + ' (' + str(bot_info.id) + ')')

    # Register Handlers
    handlers = [
        CommandHandler('start', start),
        CommandHandler('stop', stop),
        CommandHandler('help', help_text),
        CommandHandler('hilfe', help_text),
        CommandHandler('sync', run_job_manually)
    ]
    for handler in handlers:
        updater.dispatcher.add_handler(handler)

    updater.job_queue.run_repeating(
        run_job,
        interval=60.0,
        first=2.0
    )

    # Start this thing!
    updater.start_polling(
        bootstrap_retries=-1,
        allowed_updates=["message"]
    )

    # Run Bot until CTRL+C is pressed or a SIGINIT,
    # SIGTERM or SIGABRT is sent.
    updater.idle()


if __name__ == '__main__':
    main()

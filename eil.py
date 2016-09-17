#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Tagesschau-Eilmeldungen Bot
# Python 3 benötigt
#
# Freigegeben unter der GNU Affero General Public License v3.0

# Imports
import re
import time
import logging

import datetime, dateutil.parser
from requests import get
from json import loads
import redis
from configparser import ConfigParser

import telegram
from telegram.ext import Updater, Job, CommandHandler, MessageHandler, Filters
from telegram.ext.dispatcher import run_async
from telegram.error import (TelegramError, Unauthorized, BadRequest, 
                            TimedOut, NetworkError)

# Bot-Konfiguration
config = ConfigParser()
config.read_file(open('config.ini'))

redis_conf = config['REDIS']
redis_db = redis_conf.get('db' , 0)
redis_host = redis_conf.get('host')
redis_port = redis_conf.get('port', 6379)
redis_socket = redis_conf.get('socket_path')

hash = 'pythonbot:tagesschau:subs'
lhash = 'pythonbot:tagesschau:last_entry'

# Logging aktivieren und mit Redis verbinden
logging.basicConfig(format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
                    level=logging.ERROR)
logger = logging.getLogger(__name__)

if redis_socket:
    r = redis.Redis(unix_socket_path=redis_socket, db=int(redis_db))
else:
    r = redis.Redis(host=redis_host, port=int(redis_port), db=int(redis_db))

if not r.ping():
    print('Konnte nicht mit Redis verbinden, prüfe deine Einstellungen')
    quit()
 
# Kommandos
@run_async
def start(bot, update):
    if not r.sismember(hash, update.message.chat_id):
        r.sadd(hash, update.message.chat_id)
        bot.sendMessage(update.message.chat_id, '<b>Du erhältst jetzt neue Eilmeldungen!</b>\nNutze /stop, um keine Eilmeldungen mehr zu erhalten.\nFür neue Tagesschau-Artikel, check doch mal den @TagesschauDE-Kanal.', reply_to_message_id=update.message.message_id, parse_mode=telegram.ParseMode.HTML)
    else:
        bot.sendMessage(update.message.chat_id, 'Du erhältst bereits Eilmeldungen. Nutze /stop zum Deabonnieren.', reply_to_message_id=update.message.message_id)

@run_async
def stop(bot, update):
    if r.sismember(hash, update.message.chat_id):
        r.srem(hash, update.message.chat_id)
        bot.sendMessage(update.message.chat_id, '<b>Du erhältst jetzt keine Eilmeldungen mehr.</b>\nNutze /start, um wieder Eilmeldungen zu erhalten.', reply_to_message_id=update.message.message_id, parse_mode=telegram.ParseMode.HTML)
    else:
        bot.sendMessage(update.message.chat_id, 'Du hast die Eilmeldungen bereits deabonniert. Mit /start kannst du diese wieder abonnieren.', reply_to_message_id=update.message.message_id)

@run_async
def help(bot, update):
    bot.sendMessage(update.message.chat_id, '/start: Eilmeldungen erhalten\n/stop: Eilmeldungen nicht mehr erhalten.', reply_to_message_id=update.message.message_id)
    
@run_async
def run_cron(bot, job):
    last_eil = r.get(lhash)
    res = get('https://static.wiidatabase.de/personal/eil.json')
    if res.status_code != 200:
      print(time.strftime("%d.%m.%Y, %H:%M:%S") + ' Uhr: HTTP-Fehler ' + str(res.status_code))
      return
    
    data = loads(res.text)
    breakingnews = data['breakingnews']
    if not breakingnews:
      return

    if not last_eil or breakingnews[0]['date'] != last_eil.decode('utf-8'):
      title = '<b>' + breakingnews[0]['headline'] + '</b>'
      news = breakingnews[0]['shorttext'].rstrip()
      details_url = breakingnews[0]['details']
      post_url = details_url.replace('/api/', '/')
      post_url = post_url.replace('.json', '.html')
      posted_at = dateutil.parser.parse(breakingnews[0]['date'])
      posted_at = posted_at.strftime('%d.%m.%Y um %H:%M:%S Uhr')
      eilmeldung = title + '\n<i>' + posted_at + '</i>\n' + news + '\n<a href="' + post_url + '">Eilmeldung aufrufen</a>'
      r.set(lhash, breakingnews[0]['date'])
      for _, receiver in enumerate(list(r.smembers(hash))):
        try:
            bot.sendMessage(receiver.decode('utf-8'), eilmeldung, parse_mode=telegram.ParseMode.HTML, disable_web_page_preview=True)
        except Unauthorized:
            print('Chat existiert nicht mehr, lösche ihn aus der Abonnenten-Liste')
            r.srem(hash, receiver.decode('utf-8'))

def error(bot, update, error):
    logger.warn('Update "%s" verursachte Fehler "%s"' % (update, error))

def main():
    # Event-Handler
    updater = Updater(token=config['DEFAULT']['token'])
    j = updater.job_queue
    
    # Bot-Infos prüfen
    bot_info = updater.bot.getMe()
    print('Starte ' + bot_info.first_name + ', AKA @' + bot_info.username + ' (' + str(bot_info.id) + ')')

    # Handler registrieren
    dp = updater.dispatcher
    dp.add_handler(CommandHandler("start", start))
    dp.add_handler(CommandHandler("stop", stop))
    dp.add_handler(CommandHandler("help", help))
    dp.add_handler(CommandHandler("hilfe", help))
    dp.add_error_handler(error)
    
    # Prüfe auf neue Eilmeldungen
    job_minute = Job(run_cron, 60.0)
    j.put(job_minute, next_t=0.0)

    # Starte den Bot    
    updater.start_polling(clean=True)

    # Bot laufen lassen, bis CTRL+C gedrückt oder ein SIGINIT,
    # SIGTERM oder SIGABRT gesendet wird.
    updater.idle()


if __name__ == '__main__':
    main()
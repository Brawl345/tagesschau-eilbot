#!/usr/bin/env python3
# -*- coding: utf-8 -*-
import html
import logging
import re
import sys
from configparser import ConfigParser
from datetime import datetime
from json import loads

import redis
import telegram
from requests import get
from telegram.error import Unauthorized
from telegram.ext import CommandHandler, Updater
from telegram.ext.dispatcher import run_async

config = ConfigParser()
try:
    config.read_file(open("config.ini"))
except FileNotFoundError:
    logging.critical("Config.ini nicht gefunden")
    sys.exit(1)

# Logging
try:
    logging_conf = config["LOGGING"]
    logging_level = logging_conf.get("level", "INFO")
    logging_format = logging_conf.get("format", "%(asctime)s - %(levelname)s: %(message)s", raw=True)
    if logging_level not in ["DEBUG", "INFO", "CRITICAL", "ERROR", "WARNING"]:
        logging.warning("Logging Level invalid. Will be changed to WARNING")
        logging.basicConfig(format=logging_format, level=logging.INFO, datefmt="%d.%m.%Y %H:%M:%S")
    else:
        logging.basicConfig(format=logging_format,
                            level=eval("logging.{0}".format(logging_level.upper())),
                            datefmt="%d.%m.%Y %H:%M:%S")
except KeyError:
    logging.basicConfig(format="%(asctime)s - %(levelname)s: %(message)s",
                        level=logging.INFO,
                        datefmt="%d.%m.%Y %H:%M:%S")
logger = logging.getLogger(__name__)

# Bot token
try:
    bot_token = config["DEFAULT"]["token"]
except KeyError:
    logger.error("Kein Bot-Token gesetzt, bitte config.ini prüfen")
    sys.exit(1)
if not bot_token:
    logger.error("Kein Bot-Token gesetzt, bitte config.ini prüfen")
    sys.exit(1)

# Admins
try:
    admins = loads(config["ADMIN"]["id"])
except KeyError:
    logger.error("Keine Admin-IDs gesetzt, bitte config.ini prüfen.")
    admins = []
if not admins:
    logger.error("Keine Admin-IDs gesetzt, bitte config.ini prüfen.")
    admins = []

for admin in admins:
    if not isinstance(admin, int):
        logger.error("Admin-IDs müssen Integer sein.")
        admins.remove(admin)

# Redis
redis_conf = config["REDIS"]
redis_db = redis_conf.get("db", 0)
redis_host = redis_conf.get("host", "127.0.0.1")
redis_port = redis_conf.get("port", 6379)
redis_socket = redis_conf.get("socket_path")
if redis_socket:
    r = redis.Redis(unix_socket_path=redis_socket, db=int(redis_db), decode_responses=True)
else:
    r = redis.Redis(host=redis_host, port=int(redis_port), db=int(redis_db), decode_responses=True)

if not r.ping():
    logging.getLogger("Redis").critical("Redis-Verbindungsfehler, config.ini prüfen")
    sys.exit(1)

subscriber_hash = "pythonbot:tagesschau:subs"
last_entry_hash = "pythonbot:tagesschau:last_entry"


def is_group_admin(bot, update):
    res = bot.getChatMember(chat_id=update.message.chat.id, user_id=update.message.from_user.id)
    if res.status == "creator" or res.status == "administrator":
        return True
    else:
        return False


@run_async
def start(bot, update):
    if update.message.chat.type != "private":
        if not is_group_admin(bot, update):
            update.message.reply_text("❌ Nur Gruppenadministratoren können Eilmeldungen abonnieren.")
            return
    if not r.sismember(subscriber_hash, update.message.chat_id):
        r.sadd(subscriber_hash, update.message.chat_id)
        logger.info("Neuer Abonnent: " + str(update.message.chat_id))
        text = "<b>✅ Du erhältst jetzt neue Eilmeldungen!</b>\n"
        text += "Nutze /stop, um keine Eilmeldungen mehr zu erhalten.\n"
        text += "Für neue Tagesschau-Artikel, check doch mal den @TagesschauDE-Kanal.\n\n<b>ACHTUNG:</b> "
        if update.message.chat.type == "private":
            text += "Wenn du den Bot blockierst, musst du die Eilmeldungen erneut abonnieren!"
        else:
            text += "Wenn du den Bot aus der Gruppe entfernst, musst du die Eilmeldungen erneut abonnieren!"
    else:
        text = "<b>✅ Du erhältst bereits Eilmeldungen.</b>\n"
        text += "Nutze /stop zum Deabonnieren."
    update.message.reply_text(text, parse_mode=telegram.ParseMode.HTML)


@run_async
def stop(bot, update):
    if update.message.chat.type != "private":
        if not is_group_admin(bot, update):
            update.message.reply_text("❌ Nur Gruppenadministratoren können Eilmeldungen deabonnieren.")
            return
    if r.sismember(subscriber_hash, update.message.chat_id):
        r.srem(subscriber_hash, update.message.chat_id)
        logger.info("Abonnement beendet: " + str(update.message.chat_id))
        text = "<b>✅ Du erhältst jetzt keine Eilmeldungen mehr.</b>\n"
        text += "Nutze /start, um wieder Eilmeldungen zu erhalten."
    else:
        text = "<b>❌ Keine Eilmeldungen abonniert.</b>\n"
        text += "Mit /start kannst du diese abonnieren."
    update.message.reply_text(text, parse_mode=telegram.ParseMode.HTML)


@run_async
def help_text(bot, update):
    text = "/start: Eilmeldungen erhalten\n"
    text += "/stop: Eilmeldungen nicht mehr erhalten"
    update.message.reply_text(text)


@run_async
def run_job_manually(bot, update):
    if update.message.chat.id not in admins:
        return
    run_job(bot)


@run_async
def run_job(bot, job=None):
    logger.info("Prüfe auf neue Eilmeldung")
    res = get("https://www.tagesschau.de/api2/")
    if res.status_code != 200:
        logger.warning("HTTP-Fehler " + str(res.status_code))
        return

    try:
        data = loads(res.text.replace("<!-- Error -->", ""))
    except ValueError:
        logger.warning("Kein valides JSON erhalten")
        return

    if not data["news"]:
        logger.warning("Ungültiges Tagesschau-JSON")
        return

    breakingnews = data["news"][0]
    if not breakingnews["breakingNews"] and "story" not in breakingnews:
        logger.debug("Keine neue Eilmeldung")
        return

    if breakingnews["detailsweb"] == "":
        logger.warning("Keine gültige Eilmeldung erhalten")
        return

    last_breaking = r.get(last_entry_hash)
    if not last_breaking or breakingnews["externalId"] != last_breaking:
        logger.info("Neue Eilmeldung")
        title = html.escape(breakingnews["title"])
        if "content" not in breakingnews or breakingnews["content"][0]["value"] == "":
            news = ""
        else:
            news = html.escape(breakingnews["content"][0]["value"]).strip() + "\n"
        post_url = breakingnews["detailsweb"].replace("http://", "https://")
        posted_at = breakingnews["date"]
        posted_at = re.sub(r"(\+\d{2}):(\d{2})", r"\1\2", posted_at)
        posted_at = datetime.strptime(posted_at, "%Y-%m-%dT%H:%M:%S.%f%z").strftime("%d.%m.%Y um %H:%M:%S Uhr")
        text = "<b>" + title + "</b>\n"
        text += "<i>" + posted_at + "</i>\n"
        text += news
        text_link = "<a href=\"" + post_url + "\">Eilmeldung aufrufen</a>"
        reply_markup = telegram.InlineKeyboardMarkup(
            [
                [
                    telegram.InlineKeyboardButton(text="Eilmeldung aufrufen", url=post_url)
                ]
            ]
        )
        r.set(last_entry_hash, breakingnews["externalId"])
        for member in r.smembers(subscriber_hash):
            try:
                if int(member) < 0:  # Group
                    bot.sendMessage(
                        chat_id=member,
                        text="#EIL: " + text,
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
                logger.warning("Chat " + member + " existiert nicht mehr, wird gelöscht.")
                r.srem(subscriber_hash, member)
            except telegram.error.ChatMigrated as new_chat:
                new_chat_id = new_chat.new_chat_id
                logger.info("Chat migriert: " + member + " -> " + str(new_chat_id))
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
            except telegram.error.BadRequest as exception:
                logger.error(exception)


# Main function
def main():
    # Setup the updater and show bot info
    updater = Updater(token=bot_token)
    try:
        logger.info("Starte {0}, AKA @{1} ({2})".format(updater.bot.first_name, updater.bot.username, updater.bot.id))
    except Unauthorized:
        logger.critical("Anmeldung nicht möglich, Bot-Token falsch?")
        sys.exit(1)

    # Register Handlers
    handlers = [
        CommandHandler("start", start),
        CommandHandler("stop", stop),
        CommandHandler("help", help_text),
        CommandHandler("hilfe", help_text),
        CommandHandler("sync", run_job_manually)
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


if __name__ == "__main__":
    main()

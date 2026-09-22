from fastapi import FastAPI, Request, HTTPException
from telegram import Update
from telegram.ext import Application, CommandHandler, ContextTypes
import os
import yaml
import requests

app = FastAPI()

TELEGRAM_BOT_TOKEN = os.getenv("TELEGRAM_BOT_TOKEN", "")
TELEGRAM_CHAT_ID = os.getenv("TELEGRAM_CHAT_ID", "")
ALERTMANAGER_URL = os.getenv("ALERTMANAGER_URL", "http://alertmanager.monitoring.svc.cluster.local:9093")
ONCALL_SCHEDULE = "/config/oncall-schedule.yaml"

with open(ONCALL_SCHEDULE) as f:
    config = yaml.safe_load(f)
schedule = config["schedule"]
escalation_policy = config["escalation_policy"]

telegram_app = Application.builder().token(TELEGRAM_BOT_TOKEN).build()


def get_current_oncall():
    week = 1  # TODO: calculate from current date
    for entry in schedule:
        if entry["week"] == week:
            return entry
    return None


async def ack_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("✅ Alert acknowledged. On-call engineer is handling.")


async def escalate_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    oncall = get_current_oncall()
    if oncall:
        await update.message.reply_text(f"⚠️ Escalating to {oncall['tech_lead']} and {oncall['cto']}")
    else:
        await update.message.reply_text("❌ No on-call engineer found.")


async def status_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    oncall = get_current_oncall()
    if oncall:
        await update.message.reply_text(
            f"Current on-call: {oncall['engineer']}\n"
            f"Tech Lead: {oncall['tech_lead']}\n"
            f"CTO: {oncall['cto']}"
        )
    else:
        await update.message.reply_text("No on-call engineer found.")


telegram_app.add_handler(CommandHandler("ack", ack_command))
telegram_app.add_handler(CommandHandler("escalate", escalate_command))
telegram_app.add_handler(CommandHandler("status", status_command))


@app.on_event("startup")
async def startup():
    await telegram_app.initialize()
    await telegram_app.start()
    await telegram_app.updater.start_polling()


@app.on_event("shutdown")
async def shutdown():
    await telegram_app.updater.stop()
    await telegram_app.stop()
    await telegram_app.shutdown()


@app.post("/webhook/alertmanager")
async def alertmanager_webhook(request: Request):
    payload = await request.json()
    alerts = payload.get("alerts", [])
    for alert in alerts:
        message = f"🚨 <b>{alert['labels'].get('alertname', 'Alert')}</b>\n"
        message += f"Severity: {alert['labels'].get('severity', 'unknown')}\n"
        message += f"Summary: {alert['annotations'].get('summary', '')}\n"
        message += f"Description: {alert['annotations'].get('description', '')}\n"
        requests.post(
            f"https://api.telegram.org/bot{TELEGRAM_BOT_TOKEN}/sendMessage",
            json={
                "chat_id": TELEGRAM_CHAT_ID,
                "text": message,
                "parse_mode": "HTML",
            },
        )
    return {"status": "ok"}


@app.get("/health")
async def health():
    return {"status": "ok"}

from fastapi import FastAPI, Request, HTTPException  # type: ignore
from fastapi.responses import JSONResponse  # type: ignore
from telegram import Update  # type: ignore
from telegram.ext import Application, CommandHandler, CallbackQueryHandler, ContextTypes  # type: ignore
from telegram.constants import ParseMode  # type: ignore
import os
import yaml  # type: ignore
import logging
import requests  # type: ignore
from datetime import datetime, timezone
from typing import Optional

logger = logging.getLogger("oncall-bot")

app = FastAPI()

TELEGRAM_BOT_TOKEN = os.getenv("TELEGRAM_BOT_TOKEN", "")
TELEGRAM_CHAT_ID = os.getenv("TELEGRAM_CHAT_ID", "")
ALERTMANAGER_URL = os.getenv("ALERTMANAGER_URL", "http://alertmanager.monitoring.svc.cluster.local:9093")
ONCALL_SCHEDULE = "/config/oncall-schedule.yaml"
ESCALATION_ACK_TIMEOUT = int(os.getenv("ESCALATION_ACK_TIMEOUT", "300"))
ESCALATION_TIMEOUT = int(os.getenv("ESCALATION_TIMEOUT", "600"))

with open(ONCALL_SCHEDULE) as f:
    config = yaml.safe_load(f)
SCHEDULE = config["schedule"]
ESCALATION_POLICY = config["escalation-policy"]


def _get_current_week() -> int:
    now = datetime.now(timezone.utc)
    return now.isocalendar().week % 4 or 4


def get_current_oncall() -> Optional[dict]:
    week = _get_current_week()
    for entry in SCHEDULE:
        if entry["week"] == week:
            return entry
    return None


def _severity_emoji(severity: str) -> str:
    return {"critical": "🚨", "warning": "⚠️", "info": "ℹ️"}.get(severity.lower(), "🔔")


async def ack_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    oncall = get_current_oncall()
    if not oncall:
        await update.message.reply_text("❌ No on-call engineer found.")
        return
    await update.message.reply_text(
        f"✅ Alert acknowledged.\n"
        f"On-call engineer: {oncall['engineer']}\n"
        f"If not resolved within {ESCALATION_TIMEOUT}s, will escalate to {oncall['tech_lead']} → {oncall['cto']}"
    )


async def escalate_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    oncall = get_current_oncall()
    if not oncall:
        await update.message.reply_text("❌ No on-call engineer found.")
        return
    severity = context.args[0] if context.args else "sev1"
    chain = ESCALATION_POLICY.get(severity, ESCALATION_POLICY.get("sev1", []))
    names = {
        "oncall-engineer": oncall["engineer"],
        "tech-lead": oncall["tech_lead"],
        "cto": oncall["cto"],
    }
    chain_names = [names.get(role, role) for role in chain]
    await update.message.reply_text(
        f"⚠️ Escalating to: {' → '.join(chain_names)}\nSeverity: {severity}"
    )


async def status_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    oncall = get_current_oncall()
    if not oncall:
        await update.message.reply_text("No on-call engineer found.")
        return
    week = _get_current_week()
    await update.message.reply_text(
        f"Current on-call (week {week}):\n"
        f"Engineer: {oncall['engineer']}\n"
        f"Tech Lead: {oncall['tech_lead']}\n"
        f"CTO: {oncall['cto']}"
    )


async def alerts_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    try:
        response = requests.get(  # noqa: S7499
            f"{ALERTMANAGER_URL}/api/v2/alerts", timeout=5
        )
        response.raise_for_status()
        alerts = response.json()
    except Exception as e:
        await update.message.reply_text(f"❌ Failed to fetch alerts: {e}")
        return
    if not alerts:
        await update.message.reply_text("✅ No active alerts.")
        return
    lines = [f"Active alerts ({len(alerts)}):\n"]
    for alert in alerts[:10]:
        labels = alert.get("labels", {})
        annotations = alert.get("annotations", {})
        severity = labels.get("severity", "unknown")
        emoji = _severity_emoji(severity)
        name = labels.get("alertname", "Alert")
        summary = annotations.get("summary", "")
        lines.append(f"{emoji} <b>{name}</b> [{severity}]\n{summary}\n")
    await update.message.reply_text("\n".join(lines), parse_mode=ParseMode.HTML)


telegram_app = Application.builder().token(TELEGRAM_BOT_TOKEN).build()
telegram_app.add_handler(CommandHandler("ack", ack_command))
telegram_app.add_handler(CommandHandler("escalate", escalate_command))
telegram_app.add_handler(CommandHandler("status", status_command))
telegram_app.add_handler(CommandHandler("alerts", alerts_command))


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
        labels = alert.get("labels", {})
        annotations = alert.get("annotations", {})
        severity = labels.get("severity", "unknown").lower()
        alertname = labels.get("alertname", "Alert")
        summary = annotations.get("summary", "")
        description = annotations.get("description", "")
        emoji = _severity_emoji(severity)
        message = f"{emoji} <b>{alertname}</b>\nSeverity: {severity}\nSummary: {summary}\nDescription: {description}"
        try:
            requests.post(  # noqa: S7499
                f"https://api.telegram.org/bot{TELEGRAM_BOT_TOKEN}/sendMessage",
                json={
                    "chat_id": TELEGRAM_CHAT_ID,
                    "text": message,
                    "parse_mode": "HTML",
                },
                timeout=5,
            )
        except Exception:
            logger.exception("Failed to send Telegram alert")
    return {"status": "ok"}


@app.get("/health")
async def health():
    return {"status": "ok"}

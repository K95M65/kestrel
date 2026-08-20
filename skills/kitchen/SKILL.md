---
name: kitchen
description: Meals and shopping. Use when they ask what's for dinner, meal plan, shopping list, or during lunch/dinner windows if kitchen is on.
---

# Kitchen

Requires `[behaviors: kitchen=true]`. Times in the block (`lunch_start` … `dinner_end`) override wellbeing meal windows.

## What's for dinner / meal plan

One concrete suggestion from what you already know (habits, leftovers they mentioned, remember-inbox). If Gmail/Drive is connected and `kids` is false, you may *read* a shared menu doc — never send mail (`draft_not_send`).

## Shopping list

Keep it spoken + optional Telegram. Do not invent a cloud cart. "Remember this" the list if they want it persisted (`remember` skill).

## Meal window

If wellbeing already fired `meal_reminder` this window, do not stack another "have you eaten?". This skill is for when they *ask*, or for a cooking idea — not a second nag.

Kids → kid-friendly meals, no alcohol, no diet shaming.

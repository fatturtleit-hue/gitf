#!/bin/sh
# Technical Debt Report for gitf
# Generates a summary of TODO/FIXME items and overdue tasks.

echo "# Technical Debt Report - $(date +%Y-%m-%d)"
echo ""

echo "## Summary"
TODOS=$(grep -rn "TODO\|FIXME" --include="*.go" --include="*.ts" . 2>/dev/null | grep -v node_modules | grep -v .git)
COUNT=$(echo "$TODOS" | grep -c "TODO\|FIXME" 2>/dev/null || echo "0")
echo "Total TODO/FIXME items: $COUNT"
echo ""

echo "## Items"
if [ -z "$TODOS" ]; then
  echo "No TODO/FIXME items found."
else
  echo "$TODOS" | while IFS= read -r line; do
    echo "- $line"
  done
fi

echo ""
echo "## Overdue Items (past ETA date)"
TODAY=$(date +%Y-%m-%d)
echo "$TODOS" | while IFS= read -r line; do
  ETA=$(echo "$line" | grep -o 'ETA:[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}' | sed 's/ETA://')
  if [ -n "$ETA" ] && [ "$ETA" \< "$TODAY" ]; then
    echo "- $line"
  fi
done
echo ""

echo "## Debt Trend (placeholder—run periodically for history)"
echo "Date: $TODAY, Count: $COUNT"

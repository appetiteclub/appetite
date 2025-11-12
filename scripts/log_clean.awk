#!/usr/bin/awk -f

/^==> .* <==$/ {
	next
}

{
	ts = strftime("[%H:%M:%S]")
	lvl = "INF"
	msg = ""
	msgStart = 0

	for (i = 1; i <= NF; i++) {
		if ($i ~ /^level=/) {
			split($i, kv, "=")
			lvl = toupper(substr(kv[2], 1, 3))
		} else if ($i ~ /^msg=/) {
			msgStart = i
			break
		}
	}

	if (msgStart > 0) {
		for (j = msgStart; j <= NF; j++) {
			msg = msg $j " "
		}
		sub(/^msg="/, "", msg)
		sub(/"$/, "", msg)
		gsub(/\\"/, "\"", msg)
	} else {
		msg = $0
	}

	sub(/[[:space:]]+$/, "", msg)
	if (msg == "") next
	printf "%s - %s - %s\n", ts, lvl, msg
}

#!/bin/bash
sed -E \
  -e 's/^const float ([A-Za-z0-9_]+)\[([0-9]+)\]\[([0-9]+)\] = \{$/var \1 = [\2][\3]float64{/' \
  -e 's/^const float ([A-Za-z0-9_]+)\[([0-9]+)\] = \{$/var \1 = [\2]float64{/' \
  -e 's/^const int ([A-Za-z0-9_]+)\[([0-9]+)\]\[([0-9]+)\] = \{$/var \1 = [\2][\3]int{/' \
  -e 's/^const int ([A-Za-z0-9_]+)\[([0-9]+)\] = \{$/var \1 = [\2]int{/' \
  -e 's|[ 	]*//[^"]*$||' \
  -e 's/\};$/}/' "$1" \
  | awk '
    { lines[NR] = $0 }
    END {
      for (i = 1; i <= NR; i++) {
        next_line = (i < NR) ? lines[i+1] : ""
        cur = lines[i]
        # Trim trailing whitespace from cur
        sub(/[ \t]+$/, "", cur)
        # If next line is the closing brace on its own and the
        # current line ends with a digit or `}` (i.e. is an element
        # of a composite literal, not the opening line), append a
        # trailing comma.
        if (next_line ~ /^[ \t]*\}[ \t]*$/ && cur ~ /(\}|[0-9])$/) {
          cur = cur ","
        }
        print cur
      }
    }
  '

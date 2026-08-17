#!/bin/bash

for source in \
	s4d43-bibliography	\
	s4d54-transporting	\
	s4d57-implementation	\
	s4d58-sil-v3.11		\
	s4d59-terminology	\
	s4n24-errata; do

		pdftotext -layout "references/${source}.pdf" "references/${source}.txt"

done

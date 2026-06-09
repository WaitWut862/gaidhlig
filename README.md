# Passage

A linguistic research tool for Scottish Gaelic — search words, sentences, and grammar rules, with morphological analysis powered by UDPipe.

## Features

- Search across 17,000+ lemmas, 1,200 example sentences, and 343 grammar rules
- Filter results by part of speech, CEFR difficulty, and grammar category
- Token-level morphological breakdown for every sentence, powered by UDPipe and the CoNLL-U format
- Word entries include definitions, inflected forms, synonyms, derived words, etymology, and IPA pronunciation where available
- Infinite scroll for search results

## Tech Stack

- **Go** — server and routing via `net/http`
- **SQLite** — database
- **HTMX** — frontend interactivity without a JavaScript framework
- **UDPipe** — morphological analysis and CoNLL-U parsing

## Data Sources

- **kaikki.org** — Scottish Gaelic Wiktionary dump; ~17,000 lemmas with definitions, forms, IPA, and etymology
- **Dwelly's Gaelic-English Dictionary** — OCR text; ~55,000 additional entries
- **Akerbeltz.org** — 343 grammar rules across syntax, morphology, phonology, and related categories
- **USGW (Unified Scottish Gaelic Wordnet)** — synset data used for word relationships
- **scottish_gaelic-arcosg-ud-2.5** — UDPipe model used to parse 1,204 example sentences into CoNLL-U format

## Setup

1. Clone the repository
2. Ensure Go 1.22+ is installed
3. From the project root, run: `go run ./web/cmd`
4. Open `http://localhost:8080` in your browser

## Known Limitations

- Some Dwelly entries are proverbs or phrases imported as word entries due to the source format
- Other data incosistencies derived from varying data policies across sources
- The web server structure does not yet account for multiple language modules
- No user accounts or progress tracking or lessons of any form in this version

## Roadmap

- User accounts and progress tracking
- Personalized lessons and lesson plans
- CEFR difficulty tagging for more than just rules
- Expression and idiom support

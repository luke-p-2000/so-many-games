# 0004 Spike probe keeps its key in Credential Manager

Status: Superseded · the spike was removed after V1. The app uses its own entry, `SoManyGames` (requirement S2); the `SoManyGamesSpike` credential is no longer used.

The spike probe reads the Steam Web API key from Windows Credential Manager (service `SoManyGamesSpike`, user `steam-web-api-key`). The owner saves it once with `probe setkey`, which reads it with hidden input. The probe scrubs the key from every error it records, never writes it to its output files, and keeps its credential apart from the real app's so onboarding (S1) still runs from a clean state. The owner can delete it in Credential Manager after the spike.

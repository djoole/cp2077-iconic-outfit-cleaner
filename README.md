# CP2077 Iconic Outfit Cleaner

An open-source Windows utility that removes the Iconic status from selected modded Cyberpunk 2077 outfits while leaving genuine Iconic items untouched.

It scans installed TweakXL YAML files, identifies clothing records containing `Quality.IconicItem`, groups them by source mod, and lets the user choose which mods should be cleaned. Source mods are never modified.

## Requirements

- Cyberpunk 2077 for Windows
- [TweakXL](https://github.com/psiberx/cp2077-tweak-xl)
- [redscript](https://github.com/jac3km4/redscript), which is already required by TweakXL script extensions
- Microsoft Edge WebView2 Runtime, normally included with Windows 10 and 11

## Usage

1. Download `CP2077-Iconic-Outfit-Cleaner.exe` from the latest GitHub release or Nexus Mods.
2. Close Cyberpunk 2077.
3. Run the application. The game folder is used as both the scan root and patch destination by default.
4. If you use Mod Organizer 2 or another virtualized mod manager, select its mods folder as the **Mod scan root**, then select a manager-controlled mod folder as the **Patch destination**.
5. Review the detected outfit mods. All are selected by default; deselect any mod that should keep its Iconic items.
6. Select **Generate patch**, enable the generated mod in your mod manager if necessary, then launch the game normally.

The generated file is written under the selected patch destination:

```text
<patch destination>\r6\scripts\IconicOutfitCleaner\generated.reds
```

Run the cleaner again after installing or updating outfit mods.

## Features

- Detects Steam, GOG, Epic and registered Windows installations automatically.
- Recursively discovers every `r6\tweaks` folder under a user-selected scan root.
- Supports Mod Organizer 2 and other managers that keep mods outside the game directory.
- Lets the user choose a separate patch destination managed by their mod manager.
- Uses `vortex.deployment.json` when available to map every YAML file to its exact source mod.
- Selects all affected mods by default and lets the user exclude individual mods.
- Ignores weapons and other Iconic records that are not clothing.
- Expands TweakXL `$instances` templates into concrete item record IDs.
- Backs up the previous generated script before replacing it.
- Does not modify or redistribute files from Cyberpunk 2077 or third-party outfit mods.

## How it works

TweakXL merges same-level YAML array operations before applying them. Consequently, a separate YAML containing `!remove Quality.IconicItem` cannot cancel an outfit mod's `!append Quality.IconicItem` on the same record.

The cleaner therefore generates a TweakXL `ScriptableTweak`. It runs after declarative YAML imports, removes `Quality.IconicItem` from the final `statModifiers` array, and calls `TweakDBManager.UpdateRecord`. Records explicitly using `quality: Quality.Iconic` are changed to `Quality.Legendary`.

## Privacy and file access

The application does not connect to the internet, collect telemetry, launch the game, or modify outfit source files. It reads TweakXL YAML files and the optional Vortex deployment manifest. It writes only its generated script, its backups, and a small local configuration containing the last selected scan and output paths.

Configuration is stored under:

```text
%AppData%\CP2077 Iconic Outfit Cleaner\config.json
```

## Building from source

Install Go 1.22 or newer, Node.js, pnpm, and the Wails v2 CLI. Then run:

```powershell
go test ./...
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
wails build -clean
```

The executable is produced in `build/bin`. GitHub Actions also performs the same tests and build on every push and pull request.

## License

Released under the [MIT License](LICENSE).

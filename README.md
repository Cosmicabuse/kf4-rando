# King's Field IV (Ancient City) Randomizer

This is an implementation of an item randomizer for KF4, usable/tested on PCSX2 2.2.0.

## Usage

You will need :

- PCSX2 (any 2.x+ version should work, grab the latest one)
  - For settings, use defaults. Software mode renderer is blurry (like a PS2) but avoids graphical glitches
  - Vulkan seems very good at avoiding glitches as well, if you want higher native resolution
- King's Field : The Ancient City ISO (NTSC US). md5 : `e4c7bc4021db07893580f70e66cff3f6`
  - **Note**: The randomizer checks for compatible checksums on run, no need to do it yourself.
- The latest release from this repository for your system (randomizer-gui.exe for a web interface, randomizer-cli.exe for CLI with a yaml config)

Base usage setup if you don't want to bother too much :
- Open randomizer-gui.exe (download from Github Releases)
- It should open a web page. Browse your US NTSC ISO of KF4.
- Quality of Life : choose what you want to enable/disable.
- Randomizer : choose what pools should be randomized, and if you want multipliers for XP/Gold (to streamline a bit)
- Cheats : if you want to start with specific items. 
  - **IF you're not familiar with hidden walls** you can give yourself Myu's Necklace there, you can see hidden walls with it.
- Save as config.yaml : if you want to keep your settings. MIGHT BREAK ON UPDATES, if features are added.
- Run. If all worked fine, you should have a randomized ISO and logs next to the source ISO.
- To quit, exit page / exit terminal window from launching the .exe

## Features

### Randomizer

Randomized items are split in categories :
- ground items (visible and grabbable)
- containers (chest, barrel, crate, pots)
- guaranteed monster drops (optional)
- inventory's idol of sorrow
- shop items

Random monster drops are left as-is, maybe later.

The Guiding stones are NOT randomized yet.

### QoL

Some QoL patches are enabled by default, others simply available.

- Lower output ISO size : simply remove useless files. This might be bad on actual hardware, since those files are apparently designed that way to help the DVD reader read sectors "in order" or "faster".
- Remove empty containers : places empty crates/barrels/pots/stuff (there's a lot of those) out of bounds so you have less things to break, and you know easily which barrel has a check.
- Use wands anywhere : no need to step on a teleport pad to use a wand. You only need the Guiding Stone placed.
- Show item slot index in pickup text : Instead of picking up "Herb", you might pick up "Herb (#N)" with N being its index in the map. Just debug stuff, don't bother if you're not actively testing issues.
- Debug menu (L3+R3) : see the section a bit below.
- Joystick clamping : if diagonals behave weirdly (character just completely stops walking if you hold a perfect diagonal). It does require full analog stick, which is more comfortable for players not used to the control scheme.
- other QoL should be clear enough.

## Logic / Spheres

- **Endings requirements**
  - **Bad ending**  
    The randomizer logic is implemented for the good ending to make sense. You can clear without getting the Moonlight Sword, the end spheres will just make a bit less sense.
  - **Good ending**  
    You need the Moonlight Sword. To get it, you need the 4 Black Eye items as well as the Lawful Blade.  
    Use the eyes on the door at the end of the Passage of Light, and convert the Blade to the Moonlight Sword.  
    Then, kill the final boss using a final slash from the sword.

- **Names**
  - if anything (maps, monsters, etc) is named by anything, it comes from the KF4 official guide book (99% of the time at least).

- **Black Eyes**
  - I decided for now to not randomize them (they're in the 4 chests in the Holy Forest, as usual). It makes more sense to find randomized keys to unlock those, in my opinion. Maybe a setting later.

- **Binding Crown**
  - The first crown found usually in vanilla is not randomized, since you're expected to grab it and run, and logic around this is annoying (kill golem with projectile or specific conditions to grab those 2 items)

- **Other logic considerations / QoL**
  - Green and blue stones are already placed, **and using the Green Wand is expected in logic.** In vanilla, you use it after exploring all of the King's throne maps, above the ancient city.
  - projectile weapon (bow/crossbow) are weirdly needed in logic. To compensate the fact that you might need arrows, all shops sell arrows/bolts/Widda Arrows.
  - Teleporting wands can be used anywhere by default; this might break stuff (use it while falling to your death), but you do you, it's QoL, and you can disable it.

## L3+R3 Debug Menu

This was made for me to navigate the game a bit easier. L3+R3 toggles menu visibility. 
Cross to validate, Square to return back one menu. Features :

- HUD : Map ID, latest item grabbed, list of all items available (spoiled, if randomized), and their coordinates, player coordinates.
- Warp : warps you like a wand, to any map. **IMPORTANT** some maps are "one way" (you enter it, exit it, and you can't access any other map). Since the game has 2 buffers for maps, it loads the one you came from, and the one you are in. Meaning, if you warp to those maps, you will never find a loading triangle on the ground to exit it, it's broken for now.
the Warp feature needs a list of those maps with conditions ("if warping to map X, also load map Y to be able to walk out" basically).
- Levitate : a weird one, just easier to go around in the Central Tower maps. Activate it = you cannot increase your Y coordinate = you cannot fall.
Disable to start falling again - fall damage **is** calculated from where you disabled; you just lose some invisible ground under you.

## Things to know in randomizer

- **Model swapping**

Items are all modelized as a blue gem - because I felt like it, but also because getting the actual item's model is much more complex.
Hopefully that's visible enough in all circumstances.
Each map uses a # of entries for model lookups, and it's not easily changeable. For the curious:

if a map has 10*Herbs + 5*Antidotes to find, the array is size 2; if you randomize all that, you might need 15 model entries, but you really have 2.

- **The Forge's boss**

The Forge is one room in 3 states (maps 17/18/19), and its boss only exists in 18. 
When monster drops are randomized, he appears in both 18 and 19 (forge super hot, and forge regular hot) but he only drops his check in 19, so his drop isn't duplicated across states. 
With monster drops off, he stays in 18 like vanilla.

- **Starting character**

The starting character is unchanged, but can be modified via the Cheats panel in the UI.

- **Some advice**

1. If you do not want to search for fake walls all the time : give yourself **Myu's Necklace**. Wearing it makes them shine green.  
Same thing with **King's Map** : if you want a real game map, this is as close as it gets. Those items will still be randomized, but having 2 of those doesn't matter.
2. Breaking containers (crates/barrels) works with a specific formula that uses all damage types. Containers have some resistances (for ex. barrels have a 90 slash resist), which subtract from that damage type. Otherwise, all damage types are added, Physical helps.
Logic should account for this, but it's not entirely tested (and it's very annoying to do so), an automated script got what is implemented now.
3. Game is long, and can be difficult. EXP / Gold multipliers are here to help out a bit.

## Known Issues

- With ground items randomizer OFF, vanilla ground items will still show up with the gem model. Conditional model swapping is annoying.
- The Debug mode's HUD item list for monsters is wrong after loading a savefile (only kept in RAM).

## Future stuff

Paramdomizer is available but not on the GUI, I made it super early in the project and it probably needs tweaking. I'd honestly like to have other features before that I think, it was something that looked like a quick win.
If you want it you should be able to enable it in the config.yaml and specify a % range of stat modifiers. It'll randomize some stats around for weapons/armors.

Some items are not randomized, either because I didn't find how to get them, or because logic is a bit too complex for now. Or, they have weird mechanics associated to them so more reverse engineer is needed.
See [TODO.md](TODO.md) for notes of planned features if you're curious.

## Compilation

If for some reason, you need to compile it yourself :
- grab Go's latest version on [golang.org](https://golang.org)
- run `go build ./cmd/randomizer-gui/` (or `go build ./cmd/randomizer/`) for CLI version
- launch whichever executable. CLI requires a valid .yaml file, `config.example.yaml` has comments on which parameter does what. GUI is simpler.

## Bugs

Please use the github issues, giving your seed and situation.

## Contributing

If forking or whatever, please give credit. If making a PR, please :
- avoid game files (copyright stuff I guess)
- make it short if possible
- make sure CLI and GUI have feature parity.
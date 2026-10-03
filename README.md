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

## Features

Randomized items are split in categories :
- ground items (visible and grabbable)
- containers (chest, barrel, crate, pots)
- guaranteed monster drops (optional)
- inventory's idol of sorrow

Random monster drops are left as-is, maybe later. The Fire Lord's Fire Mail is the exception, it counts as a guaranteed drop.

## Logic / Spheres

- **Endings requirements**
  - **Bad ending**  
    The randomizer logic is implemented for the good ending to make sense. You can clear without getting the Moonlight Sword, the end spheres will just make a bit less sense.
  - **Good ending**  
    You need the Moonlight Sword. To get it, you need the 4 Black Eye items as well as the Lawful Blade.  
    Use the eyes on the door at the end of the Passage of Light, and convert the Blade to the Moonlight Sword.  
    Then, kill the final boss using a final slash from the sword.

- **Black Eyes**
  - I decided for now to not randomize them (they're in the 4 chests in the Holy Forest, as usual). It makes more sense to find randomized keys to unlock those, in my opinion. Maybe a setting later.

- **Sphere requirements**
  - Without randomizing, the usual progression I've noted for now in the logic is :
    - maps 0, 2, 4, 8, 9, 29, 96, 98, 99 require nothing (the well is considered free, you need 130+ HP to survive the fall which is hard to modelize in logic)
    - maps 5, 11, 12, 95, 97 require Fire Crystal - Map 2A requires Rusty Key (house in map 2)
    - maps 6, 7, 13-15, 16B

## Things to know in randomizer

- **Model swapping**

Items are all modelized as a blue gem - because I felt like it, but also because getting the actual item's model is much more complex.
Hopefully that's visible enough in all circumstances.
Each map uses a # of entries for model lookups, and it's not easily changeable. For the curious:

if a map has 10*Herbs + 5*Antidotes to find, the array is size 2; if you randomize all that, you might need 15 model entries, but you really have 2.

- **The Forge's boss**

The Forge is one room in 3 states (maps 17/18/19), and its boss only exists in 18. When monster drops are randomized, he's moved to 19 along with his loot, so his drop isn't duplicated across states. With monster drops off, he stays in 18 like vanilla.

- **Starting character**

The starting character is unchanged, but can be modified via the Cheats panel in the UI.

- **Some advice**

1. The game is hard, and takes a while to get started (because of Physical and Magical being low, and MP being low). Knowing this, I strongly advise you to give yourself
more starting stats if you want a "short enough" rando.
2. If you do not want to search for fake walls all the time : give yourself **Myu's Necklace**. Wearing it makes them shine green.  
Same thing with **King's Map** : if you want a real game map, this is as close as it gets. Those items will still be randomized, but having 2 of those doesn't matter.
3. Breaking containers (crates/barrels) works with a specific formula that uses all damage types. Containers have some resistances (for ex. barrels have a 90 slash resist), which subtract from that damage type. Otherwise, all damage types are added, Physical helps
4. QoL patches :
   - full analog controls helps with comfort (if you have a bad left hand, which I do)
   - enable joystick clamping if diagonals behave weirdly (character just completely stops walking if you hold a perfect diagonal)
5. 

## Future stuff

See [TODO.md](TODO.md) for notes of planned features if you're curious.

## Compilation

If for some reason, you need to compile it yourself :
- grab Go's latest version on [golang.org](https://golang.org)
- run `go build ./cmd/randomizer-gui/` (or `go build ./cmd/randomizer/`) for CLI version
- launch whichever executable. CLI requires a valid .yaml file, `config.example.yaml` has comments on which parameter does what. I strongly advice GUI, it's 1:1 on features.

## Bugs

Please use the github issues, giving your seed and situation.

## Contributing

If forking or whatever, please give credit. If making a PR, please :
- avoid game files (copyright stuff I guess)
- make it short if possible
- make sure CLI and GUI have feature parity.
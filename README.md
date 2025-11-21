# Appetite

![Appetite Orders Board](docs/img/gallery/orders-board.png)
<p align="right"><small><a href="docs/gallery/index.md">See more in the gallery</a></small></p>

I wanted to explore an alternative way to build a restaurant venue manager while also starting a new iteration of [Aquamarine](https://github.com/aquamarinepk), this time based on [aqm](https://github.com/aquamarinepk/aqm) lib. And here we are.

---

## Installation

### Clone
```bash
git clone https://github.com/appetiteclub/appetite.git
cd appetite
```

### Run
To quickly start all available services locally:
```bash
make run-all
```

This command builds all components and starts the local stack (Admin, AuthN, AuthZ, Table).
Logs for each service are stored in their respective folders under `services/`.

On the first run, the console will display the generated user and password to access the interface and explore the current implementation.

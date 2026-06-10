import random
import time

PROFILES = {
    "fast": (0.008, 0.035),
    "normal": (0.04, 0.12),
    "db": (0.06, 0.22),
    "slow": (0.25, 0.9),
    "spike": (1.0, 2.8),
    "flash": (0.15, 0.55),
}


def simulate(profile="normal", spike_chance=0.0, spike_profile="spike"):
    low, high = PROFILES.get(profile, PROFILES["normal"])
    if spike_chance > 0 and random.random() < spike_chance:
        low, high = PROFILES[spike_profile]
    time.sleep(random.uniform(low, high))


def maybe_fail(rate=0.0):
    return random.random() < rate

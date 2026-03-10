MAX_RETRIES = 3


class Animal:
    """Animal is the base class for all animals."""

    def __init__(self, name):
        """__init__ initializes a new Animal instance."""
        self.name = name

    def speak(self):
        """speak outputs the animal's sound."""
        print(self.name)


def greet(name):
    """greet creates a greeting string for the given name."""
    return f"Hello, {name}"


def helper(x, y):
    """helper adds two numbers."""
    return x + y

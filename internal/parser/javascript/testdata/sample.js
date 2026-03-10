// MaxRetries is the maximum retry count.
const MaxRetries = 3;

// Animal is the base class for all animals.
class Animal {
  constructor(name) {
    this.name = name;
  }

  // speak outputs the animal's sound.
  speak() {
    console.log(this.name);
  }
}

// greet creates a greeting string for the given name.
function greet(name) {
  return `Hello, ${name}`;
}

// helper is a utility function.
function helper(x, y) {
  return x + y;
}

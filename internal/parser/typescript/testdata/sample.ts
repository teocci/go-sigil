// MaxRetries is the maximum number of retry attempts.
const MaxRetries = 3;

// Greeter interface defines objects that can greet.
interface Greeter {
  greet(name: string): string;
}

// Point is a 2D coordinate type.
type Point = { x: number; y: number };

// Animal is the base class for all animals.
class Animal {
  name: string;

  constructor(name: string) {
    this.name = name;
  }

  // speak outputs the animal's sound.
  speak(): void {
    console.log(this.name);
  }
}

// greet creates a greeting string for the given name.
function greet(name: string): string {
  return `Hello, ${name}`;
}

// helper is an unexported utility function.
function helper(x: number, y: number): number {
  return x + y;
}

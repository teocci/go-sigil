// MaxRetries is the maximum number of retry attempts.
const MAX_RETRIES: u32 = 3;

// WorkerError describes errors produced by the worker.
type WorkerError = Box<dyn std::error::Error>;

/// Processor defines a processing contract.
pub trait Processor {
    fn process(&self, data: &[u8]) -> Result<(), WorkerError>;
}

/// Worker performs background work.
pub struct Worker {
    name: String,
}

impl Worker {
    /// new creates a Worker with the given name.
    pub fn new(name: &str) -> Self {
        Worker { name: name.to_string() }
    }

    /// run starts the worker processing loop.
    pub fn run(&self) {
        println!("{}", self.name);
        let _ = helper(1, 2);
    }
}

/// Status enumerates possible worker states.
pub enum Status {
    Idle,
    Running,
    Done,
}

/// helper is an unexported helper function.
fn helper(x: i32, y: i32) -> i32 {
    x + y
}

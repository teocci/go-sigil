package com.example;

/**
 * MAX_RETRIES is the maximum number of retry attempts.
 */
public class Sample {

    /** The maximum number of retry attempts. */
    public static final int MAX_RETRIES = 3;

    private String name;

    /** Creates a Sample with the given name. */
    public Sample(String name) {
        this.name = name;
    }

    /** Returns the name. */
    public String getName() {
        return this.name;
    }

    /** Runs the sample logic. */
    public void run() {
        System.out.println(this.name);
    }

    /** helper is an internal helper method. */
    private static int helper(int x, int y) {
        return x + y;
    }
}

/** Processor defines a processing contract. */
interface Processor {
    void process(byte[] data) throws Exception;
}

/** Status enumerates possible states. */
enum Status {
    IDLE,
    RUNNING,
    DONE;
}

fn main() {
    cxx_build::bridge("src/lib.rs") // Path to your lib.rs file
        .flag_if_supported("-std=c++14") // Specify C++ standard
        .compile("my_rust_library");

    println!("cargo:rerun-if-changed=src/lib.rs");
}

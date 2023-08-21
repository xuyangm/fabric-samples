def create_text_file(file_name, size_in_bytes):
    with open(file_name, 'w') as f:
        # Set the chunk size for writing data
        chunk_size = 1024  # You can adjust this based on your needs
        
        # Calculate the number of iterations needed to reach the desired size
        iterations = size_in_bytes // chunk_size
        
        # Create a chunk of data
        data_chunk = 'A' * chunk_size
        
        # Write the chunk to the file for the specified number of iterations
        for _ in range(iterations):
            f.write(data_chunk)
        
        # Write any remaining bytes (if size_in_bytes is not a multiple of chunk_size)
        remaining_bytes = size_in_bytes % chunk_size
        if remaining_bytes > 0:
            f.write('A' * remaining_bytes)

if __name__ == "__main__":
    file_name = "output.txt"
    size_in_bytes = int(input("Enter the size of the file in bytes: "))
    create_text_file(file_name, size_in_bytes)
    print(f"File '{file_name}' created with a size of {size_in_bytes} bytes.")
import random
import argparse
import string

def create_random_text_file(file_name, size_in_chars):
    with open(file_name, 'w') as f:
        random_data = ''.join(random.choice(string.ascii_letters+string.digits) for _ in range(size_in_chars))
        f.write(random_data)

def main():
    parser = argparse.ArgumentParser(description='Create a text file filled with random data.')
    parser.add_argument('file_name', type=str, help='Name of the output file')
    parser.add_argument('size_in_chars', type=int, help='Number of characters for the file')

    args = parser.parse_args()

    create_random_text_file(args.file_name, args.size_in_chars)
    print(f"File '{args.file_name}' created with random data and a size of {args.size_in_chars} characters.")

if __name__ == "__main__":
    main()
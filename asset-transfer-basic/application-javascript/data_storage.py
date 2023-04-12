import json

class Chunk:
    def __init__(self):
        self.chunk_hash = None
        
    def set_hash_value(self, hash_value):
        self.chunk_hash = hash_value
        
    def to_json(self):
        return json.dumps({'chunkHash': self.chunk_hash}, indent=4)

class Stripe:
    def __init__(self):
        self.hash_value = None
        self.chunks = []  # list of Chunk objects
        
    def set_hash_value(self, hash_value):
        self.hash_value = hash_value
        
    def add_chunk(self, chunk):
        self.chunks.append(chunk)
        
    def to_json(self):
        chunk_list = []
        for chunk in self.chunks:
            chunk_list.append(json.loads(chunk.to_json()))
        return json.dumps({'stripeHash': self.hash_value, 'chunkHashes': chunk_list}, indent=4)

class File:
    def __init__(self):
        self.hash_value = None
        self.stripes = []  # list of Stripe objects
        
    def set_hash_value(self, hash_value):
        self.hash_value = hash_value
        
    def add_stripe(self, stripe):
        self.stripes.append(stripe)
        
    def to_json(self):
        stripe_list = []
        for stripe in self.stripes:
            stripe_list.append(json.loads(stripe.to_json()))
        return json.dumps({'stripeHashes': stripe_list}, indent=4)
    
    def output_to_json_file(self):
        file_name = self.hash_value + '.json'
        with open(file_name, 'w') as f:
            f.write(self.to_json())


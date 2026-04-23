import { source } from '@/lib/source';
import { createFromSource } from 'fumadocs-core/search/server';
import { createTokenizer } from '@orama/tokenizers/mandarin';

export const revalidate = false;

const tokenizer = await createTokenizer();

export const { staticGET: GET } = createFromSource(source, {
  tokenizer,
});

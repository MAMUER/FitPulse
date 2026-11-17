import { describe, expect, it } from 'vitest';
import { isValidPathSegment, pathSegment } from './pathSegment';

describe('pathSegment', () => {
  it('accepts numeric, string and uuid identifiers', () => {
    expect(pathSegment(10)).toBe('10');
    expect(pathSegment('e1')).toBe('e1');
    expect(pathSegment('550e8400-e29b-41d4-a716-446655440000')).toBe(
      '550e8400-e29b-41d4-a716-446655440000'
    );
    expect(pathSegment('a.b_c~d')).toBe('a.b_c~d');
  });

  it('rejects empty and missing values', () => {
    expect(() => pathSegment('')).toThrow('Invalid path parameter');
    expect(() => pathSegment(null)).toThrow('Invalid path parameter');
    expect(() => pathSegment(undefined)).toThrow('Invalid path parameter');
  });

  it('rejects path traversal and path separators', () => {
    expect(() => pathSegment('.')).toThrow('Invalid path parameter');
    expect(() => pathSegment('..')).toThrow('Invalid path parameter');
    expect(() => pathSegment('../admin')).toThrow('Invalid path parameter');
    expect(() => pathSegment('a/b')).toThrow('Invalid path parameter');
    expect(() => pathSegment('a\\b')).toThrow('Invalid path parameter');
    expect(() => pathSegment('.hidden')).toThrow('Invalid path parameter');
  });

  it('rejects URLs, protocol-relative values, query and fragment characters', () => {
    expect(() => pathSegment('//evil.example')).toThrow(
      'Invalid path parameter'
    );
    expect(() => pathSegment('https://evil.example')).toThrow(
      'Invalid path parameter'
    );
    expect(() => pathSegment('a?b=1')).toThrow('Invalid path parameter');
    expect(() => pathSegment('a#fragment')).toThrow('Invalid path parameter');
    expect(() => pathSegment('a b')).toThrow('Invalid path parameter');
  });

  it('rejects overlong identifiers', () => {
    expect(() => pathSegment('a'.repeat(129))).toThrow(
      'Invalid path parameter'
    );
  });

  it('reports validity without throwing', () => {
    expect(isValidPathSegment(10)).toBe(true);
    expect(isValidPathSegment('e1')).toBe(true);
    expect(isValidPathSegment('../admin')).toBe(false);
    expect(isValidPathSegment(null)).toBe(false);
  });
});

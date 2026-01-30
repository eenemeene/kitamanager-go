'use client';

import * as React from 'react';
import { X, Plus } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';

export interface FundingAttribute {
  key: string;
  value: string;
}

/**
 * Scalar-only contract properties for the tag input.
 * This is a subset of ContractProperties that only allows string values (not arrays).
 */
export type ScalarContractProperties = Record<string, string>;

export interface PropertyTagInputProps {
  /** Current properties as key-value pairs (scalar values only) */
  value: ScalarContractProperties | undefined;
  /** Called when properties change */
  onChange: (value: ScalarContractProperties | undefined) => void;
  /** Available funding attributes with their keys */
  fundingAttributes?: FundingAttribute[];
  /** Attributes grouped by key for display */
  attributesByKey?: Record<string, FundingAttribute[]>;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  id?: string;
  suggestionsLabel?: string;
}

/**
 * Tag input component for contract properties.
 * Handles key-based exclusivity: selecting a value replaces any existing value with the same key.
 */
export function PropertyTagInput({
  value,
  onChange,
  fundingAttributes = [],
  attributesByKey = {},
  placeholder = 'Select attributes...',
  disabled = false,
  className,
  id,
  suggestionsLabel,
}: PropertyTagInputProps) {
  // Get currently selected values
  const selectedValues = value ? Object.values(value) : [];

  // Add an attribute (replaces any existing value with the same key)
  const addAttribute = (attr: FundingAttribute) => {
    const newProps = { ...value, [attr.key]: attr.value };
    onChange(newProps);
  };

  // Remove an attribute by its value
  const removeAttribute = (valueToRemove: string) => {
    if (!value) return;
    const newProps: ScalarContractProperties = {};
    for (const [k, v] of Object.entries(value)) {
      if (v !== valueToRemove) {
        newProps[k] = v;
      }
    }
    onChange(Object.keys(newProps).length > 0 ? newProps : undefined);
  };

  // Get the key for a selected value
  const getKeyForValue = (val: string): string | undefined => {
    if (!value) return undefined;
    for (const [k, v] of Object.entries(value)) {
      if (v === val) return k;
    }
    return undefined;
  };

  // Filter suggestions to show available ones
  // A suggestion is available if:
  // 1. It's not already selected, OR
  // 2. Another value with the same key is selected (it would replace it)
  const getAvailableSuggestions = () => {
    return fundingAttributes.filter((attr) => {
      // Already selected - don't show
      if (selectedValues.includes(attr.value)) return false;
      return true;
    });
  };

  const availableSuggestions = getAvailableSuggestions();

  // Check if selecting this attribute would replace another
  const wouldReplace = (attr: FundingAttribute): string | undefined => {
    if (!value) return undefined;
    const existingValue = value[attr.key];
    if (existingValue && typeof existingValue === 'string' && existingValue !== attr.value) {
      return existingValue;
    }
    return undefined;
  };

  return (
    <div className="space-y-2">
      <div
        className={cn(
          'flex min-h-10 w-full flex-wrap items-center gap-1.5 rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-2',
          disabled && 'cursor-not-allowed opacity-50',
          className
        )}
      >
        {selectedValues.length === 0 && (
          <span className="text-muted-foreground">{placeholder}</span>
        )}
        {selectedValues.map((val) => {
          const key = getKeyForValue(val);
          return (
            <Badge
              key={val}
              variant="secondary"
              className="gap-1 pr-1"
              title={key ? `Key: ${key}` : undefined}
            >
              {val}
              {!disabled && (
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    removeAttribute(val);
                  }}
                  className="ml-1 rounded-full outline-none ring-offset-background hover:bg-muted focus:ring-2 focus:ring-ring focus:ring-offset-2"
                  aria-label={`Remove ${val}`}
                >
                  <X className="h-3 w-3" />
                </button>
              )}
            </Badge>
          );
        })}
      </div>
      {availableSuggestions.length > 0 && !disabled && (
        <div className="flex flex-wrap gap-1">
          {suggestionsLabel && (
            <span className="mr-1 self-center text-xs text-muted-foreground">
              {suggestionsLabel}
            </span>
          )}
          {availableSuggestions.map((attr) => {
            const replaces = wouldReplace(attr);
            return (
              <button
                key={attr.value}
                type="button"
                onClick={() => addAttribute(attr)}
                className={cn(
                  'inline-flex items-center gap-1 rounded-full border border-dashed px-2 py-0.5 text-xs transition-colors',
                  replaces
                    ? 'border-orange-500/50 text-orange-600 hover:border-orange-500 hover:bg-orange-50'
                    : 'border-muted-foreground/50 text-muted-foreground hover:border-primary hover:text-primary'
                )}
                title={replaces ? `Replaces: ${replaces}` : `Key: ${attr.key}`}
              >
                <Plus className="h-3 w-3" />
                {attr.value}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}

// Keep the simple TagInput for backwards compatibility
export interface TagInputProps {
  value: string[];
  onChange: (value: string[]) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  id?: string;
  suggestions?: string[];
  suggestionsLabel?: string;
}

export function TagInput({
  value,
  onChange,
  placeholder = 'Type and press Enter...',
  disabled = false,
  className,
  id,
  suggestions = [],
  suggestionsLabel,
}: TagInputProps) {
  const [inputValue, setInputValue] = React.useState('');
  const inputRef = React.useRef<HTMLInputElement>(null);

  const addTag = (tag: string) => {
    const trimmed = tag.trim().toLowerCase();
    if (!trimmed || value.includes(trimmed)) {
      setInputValue('');
      return;
    }

    onChange([...value, trimmed]);
    setInputValue('');
  };

  const removeTag = (tagToRemove: string) => {
    onChange(value.filter((tag) => tag !== tagToRemove));
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault();
      if (inputValue.trim()) {
        addTag(inputValue);
      }
    } else if (e.key === 'Backspace' && !inputValue && value.length > 0) {
      removeTag(value[value.length - 1]);
    }
  };

  const handleBlur = () => {
    if (inputValue.trim()) {
      addTag(inputValue);
    }
  };

  const availableSuggestions = suggestions.filter((s) => !value.includes(s.toLowerCase()));

  return (
    <div className="space-y-2">
      <div
        className={cn(
          'flex min-h-10 w-full flex-wrap items-center gap-1.5 rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-2',
          disabled && 'cursor-not-allowed opacity-50',
          className
        )}
        onClick={() => inputRef.current?.focus()}
      >
        {value.map((tag) => (
          <Badge key={tag} variant="secondary" className="gap-1 pr-1">
            {tag}
            {!disabled && (
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  removeTag(tag);
                }}
                className="ml-1 rounded-full outline-none ring-offset-background hover:bg-muted focus:ring-2 focus:ring-ring focus:ring-offset-2"
                aria-label={`Remove ${tag}`}
              >
                <X className="h-3 w-3" />
              </button>
            )}
          </Badge>
        ))}
        <input
          ref={inputRef}
          id={id}
          type="text"
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          onKeyDown={handleKeyDown}
          onBlur={handleBlur}
          placeholder={value.length === 0 ? placeholder : ''}
          disabled={disabled}
          className="flex-1 bg-transparent outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed"
          style={{ minWidth: '80px' }}
        />
      </div>
      {availableSuggestions.length > 0 && !disabled && (
        <div className="flex flex-wrap gap-1">
          {suggestionsLabel && (
            <span className="mr-1 self-center text-xs text-muted-foreground">
              {suggestionsLabel}
            </span>
          )}
          {availableSuggestions.map((suggestion) => (
            <button
              key={suggestion}
              type="button"
              onClick={() => addTag(suggestion)}
              className="inline-flex items-center gap-1 rounded-full border border-dashed border-muted-foreground/50 px-2 py-0.5 text-xs text-muted-foreground transition-colors hover:border-primary hover:text-primary"
            >
              <Plus className="h-3 w-3" />
              {suggestion}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

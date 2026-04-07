import type { ContractProperties } from '@/lib/api/types';

/**
 * Contract properties utilities
 * Properties are stored as {key: value} where key is the category
 * and value is the selected option. Keys are mutually exclusive
 * (e.g., care_type can only have one value).
 * Values can be strings (scalar) or string arrays.
 */

export type { ContractProperties };

export interface FundingAttribute {
  key: string;
  value: string;
}

/**
 * Convert contract properties map to a flat array of values.
 * Used for displaying selected attributes in the UI.
 * Handles both scalar strings and string arrays.
 */
export function propertiesToValues(properties?: ContractProperties): string[] {
  if (!properties) return [];
  const result: string[] = [];
  for (const value of Object.values(properties)) {
    if (Array.isArray(value)) {
      result.push(...value);
    } else {
      result.push(value);
    }
  }
  return result;
}

/**
 * Get the value for a specific key from properties.
 * Returns the value if it's a string, or undefined if it's an array or doesn't exist.
 */
export function getPropertyValue(
  properties: ContractProperties | undefined,
  key: string
): string | string[] | undefined {
  return properties?.[key];
}

/**
 * Get scalar value for a specific key from properties.
 * Returns the value only if it's a string, not an array.
 */
export function getScalarPropertyValue(
  properties: ContractProperties | undefined,
  key: string
): string | undefined {
  const value = properties?.[key];
  return typeof value === 'string' ? value : undefined;
}

/**
 * Set a property value. If the key already exists, it replaces the value.
 * Returns a new properties object.
 */
export function setProperty(
  properties: ContractProperties | undefined,
  key: string,
  value: string | string[]
): ContractProperties {
  return { ...properties, [key]: value };
}

/**
 * Remove a property by its value. Finds the key that has this value and removes it.
 * Returns a new properties object.
 */
export function removePropertyByValue(
  properties: ContractProperties | undefined,
  value: string
): ContractProperties | undefined {
  if (!properties) return undefined;

  const newProps: ContractProperties = {};
  for (const [k, v] of Object.entries(properties)) {
    if (v !== value) {
      newProps[k] = v;
    }
  }
  return Object.keys(newProps).length > 0 ? newProps : undefined;
}

/**
 * Check if properties contain a specific value.
 */
export function hasPropertyValue(
  properties: ContractProperties | undefined,
  value: string
): boolean {
  if (!properties) return false;
  return Object.values(properties).includes(value);
}

/**
 * Get the key for a value in properties.
 */
export function getKeyForValue(
  properties: ContractProperties | undefined,
  value: string
): string | undefined {
  if (!properties) return undefined;
  for (const [k, v] of Object.entries(properties)) {
    if (v === value) return k;
  }
  return undefined;
}

/**
 * Convert contract properties to key--value label keys for i18n lookup.
 * E.g., {care_type: "ganztag", ndh: "ndh"} → ["care_type--ganztag", "ndh--ndh"]
 * Handles both scalar strings and string arrays.
 */
export function propertiesToLabelKeys(properties?: ContractProperties): string[] {
  if (!properties) return [];
  const result: string[] = [];
  for (const [key, value] of Object.entries(properties)) {
    if (Array.isArray(value)) {
      for (const v of value) {
        result.push(`${key}--${v}`);
      }
    } else {
      result.push(`${key}--${value}`);
    }
  }
  return result;
}

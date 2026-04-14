import { AbstractControl, ValidationErrors } from '@angular/forms';

const LATIN_PATTERN = /^[a-zA-Z\u00C0-\u024F\d\s.,\-'\/()&#:;@+]*$/;

export function latinOnly(control: AbstractControl): ValidationErrors | null {
  const value: string = control.value;
  if (!value) {
    return null;
  }
  return LATIN_PATTERN.test(value) ? null : { latinOnly: true };
}

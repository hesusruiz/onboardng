import { Component, ViewEncapsulation, ElementRef, EventEmitter, Input, Output, QueryList, ViewChildren } from '@angular/core';
import { NgClass } from '@angular/common';

@Component({
  selector: 'app-email-verification',
  standalone: true,
  imports: [NgClass],
  templateUrl: './email-verification.component.html',
  styleUrl: './email-verification.component.css',
  encapsulation: ViewEncapsulation.None
})
export class EmailVerificationComponent {
  @Input() email = '';
  @Input() isLoading = false;
  @Input() errorMessage = '';
  @Output() verify = new EventEmitter<string>();
  @Output() resend = new EventEmitter<void>();

  @ViewChildren('otpInput') otpInputs!: QueryList<ElementRef>;

  otpDigits: string[] = ['', '', '', '', '', ''];

  get otpCode(): string {
    return this.otpDigits.join('');
  }

  get isComplete(): boolean {
    return this.otpDigits.every(d => d !== '');
  }

  onInput(index: number, event: Event): void {
    const input = event.target as HTMLInputElement;
    const value = input.value;

    if (value.length > 1) {
      this.otpDigits[index] = value.charAt(value.length - 1);
      input.value = this.otpDigits[index];
    } else {
      this.otpDigits[index] = value;
    }

    if (value && index < 5) {
      const inputs = this.otpInputs.toArray();
      inputs[index + 1].nativeElement.focus();
    }
  }

  onKeydown(index: number, event: KeyboardEvent): void {
    if (event.key === 'Backspace' && !this.otpDigits[index] && index > 0) {
      const inputs = this.otpInputs.toArray();
      inputs[index - 1].nativeElement.focus();
    }
  }

  onFocus(index: number, event: FocusEvent): void {
    const firstEmpty = this.otpDigits.findIndex(d => d === '');
    const allowedIndex = firstEmpty === -1 ? this.otpDigits.length - 1 : firstEmpty;

    if (index > allowedIndex) {
      event.preventDefault();
      const inputs = this.otpInputs.toArray();
      inputs[allowedIndex].nativeElement.focus();
    }
  }

  onPaste(event: ClipboardEvent): void {
    event.preventDefault();
    const pasted = (event.clipboardData?.getData('text') || '').replace(/\D/g, '').slice(0, 6);
    const inputs = this.otpInputs.toArray();

    for (let i = 0; i < pasted.length; i++) {
      this.otpDigits[i] = pasted[i];
      inputs[i].nativeElement.value = pasted[i];
    }

    const nextIndex = Math.min(pasted.length, 5);
    inputs[nextIndex].nativeElement.focus();
  }

  onVerify(): void {
    if (this.isComplete) {
      this.verify.emit(this.otpCode);
    }
  }

  onResend(): void {
    this.otpDigits = ['', '', '', '', '', ''];
    const inputs = this.otpInputs.toArray();
    inputs.forEach(input => input.nativeElement.value = '');
    inputs[0].nativeElement.focus();
    this.resend.emit();
  }
}

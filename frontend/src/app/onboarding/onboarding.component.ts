import { Component, ViewEncapsulation, inject, ChangeDetectorRef } from '@angular/core';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { LandingComponent } from './components/landing/landing.component';
import { TermsComponent } from './components/terms/terms.component';
import { RepresentativeComponent } from './components/representative/representative.component';
import { CompanyInfoComponent } from './components/company-info/company-info.component';
import { EmailVerificationComponent } from './components/email-verification/email-verification.component';
import { SuccessComponent } from './components/success/success.component';

@Component({
  selector: 'app-onboarding',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    LandingComponent,
    TermsComponent,
    RepresentativeComponent,
    CompanyInfoComponent,
    EmailVerificationComponent,
    SuccessComponent
  ],
  templateUrl: './onboarding.component.html',
  styleUrl: './onboarding.component.css',
  encapsulation: ViewEncapsulation.None
})
export class OnboardingComponent {
  private http = inject(HttpClient);
  private cdr = inject(ChangeDetectorRef);

  currentStep = 0;
  isValidatingEmail = false;
  emailError = '';
  isVerifyingCode = false;
  verifyError = '';
  isRegistering = false;
  registerError = '';
  verifiedCode = '';

  onboardingForm = new FormGroup({
    isCompany: new FormControl(false, [Validators.requiredTrue]),
    acceptTerms: new FormControl(false, [Validators.requiredTrue]),
    representative: new FormGroup({
      firstName: new FormControl('', [Validators.required]),
      lastName: new FormControl('', [Validators.required]),
      email: new FormControl('', [Validators.required, Validators.email]),
      isAuthorised: new FormControl(false, [Validators.requiredTrue]),
    }),
    company: new FormGroup({
      legalName: new FormControl('', [Validators.required]),
      vatNumber: new FormControl('', [Validators.required]),
      country: new FormControl('', [Validators.required]),
      city: new FormControl('', [Validators.required]),
      street: new FormControl('', [Validators.required]),
      postalCode: new FormControl('', [Validators.required]),
    }),
  });

  get isCompanyControl(): FormControl {
    return this.onboardingForm.get('isCompany') as FormControl;
  }

  get acceptTermsControl(): FormControl {
    return this.onboardingForm.get('acceptTerms') as FormControl;
  }

  get representativeGroup(): FormGroup {
    return this.onboardingForm.get('representative') as FormGroup;
  }

  get companyGroup(): FormGroup {
    return this.onboardingForm.get('company') as FormGroup;
  }

  goNext(): void {
    this.currentStep++;
  }

  goBack(): void {
    this.currentStep--;
  }

  validateEmailAndNext(): void {
    const email = this.representativeGroup.get('email')?.value;
    this.isValidatingEmail = true;
    this.emailError = '';

    const headers = new HttpHeaders({
      'Content-Type': 'application/json',
      'X-Requested-With': 'XMLHttpRequest'
    });

    this.http.post<any>('/api/validate-email', { email }, { headers })
      .subscribe({
        next: (res: any) => {
          this.isValidatingEmail = false;
          if (res && res.success === true) {
            this.currentStep++;
          } else {
            this.emailError = res?.message || 'Failed to validate email. Please try again.';
          }
          this.cdr.detectChanges();
        },
        error: (err: any) => {
          this.isValidatingEmail = false;
          this.emailError = err.error?.message || 'Failed to validate email. Please try again.';
          this.cdr.detectChanges();
        }
      });
  }

  onVerify(code: string): void {
    const email = this.representativeGroup.get('email')?.value;
    this.isVerifyingCode = true;
    this.verifyError = '';

    const headers = new HttpHeaders({
      'Content-Type': 'application/json',
      'X-Requested-With': 'XMLHttpRequest'
    });

    this.http.post<any>('/api/verify-code', { email, code }, { headers })
      .subscribe({
        next: (res: any) => {
          this.isVerifyingCode = false;
          if (res && res.success === true) {
            this.verifiedCode = code;
            this.currentStep = 4;
          } else {
            this.verifyError = res?.message || 'Invalid verification code. Please try again.';
          }
          this.cdr.detectChanges();
        },
        error: (err: any) => {
          this.isVerifyingCode = false;
          this.verifyError = err.error?.message || 'Invalid verification code. Please try again.';
          this.cdr.detectChanges();
        }
      });
  }

  onResendCode(): void {
    const email = this.representativeGroup.get('email')?.value;
    this.verifyError = '';

    const headers = new HttpHeaders({
      'Content-Type': 'application/json',
      'X-Requested-With': 'XMLHttpRequest'
    });

    this.http.post<any>('/api/validate-email', { email }, { headers })
      .subscribe({
        next: () => this.cdr.detectChanges(),
        error: (err: any) => {
          this.verifyError = err.error?.message || 'Failed to resend code. Please try again.';
          this.cdr.detectChanges();
        }
      });
  }

  onComplete(): void {
    const rep = this.representativeGroup.value;
    const company = this.companyGroup.value;
    this.isRegistering = true;
    this.registerError = '';

    const headers = new HttpHeaders({
      'Content-Type': 'application/json',
      'X-Requested-With': 'XMLHttpRequest'
    });

    const body = {
      firstName: rep.firstName,
      lastName: rep.lastName,
      email: rep.email,
      companyName: company.legalName,
      vatId: company.vatNumber,
      country: company.country,
      streetAddress: company.street,
      postalCode: company.postalCode,
      code: this.verifiedCode,
    };

    this.http.post<any>('/api/register', body, { headers })
      .subscribe({
        next: (res: any) => {
          this.isRegistering = false;
          if (res && res.success === true) {
            this.currentStep = 5;
          } else {
            this.registerError = res?.message || 'Registration failed. Please try again.';
          }
          this.cdr.detectChanges();
        },
        error: (err: any) => {
          this.isRegistering = false;
          this.registerError = err.error?.message || 'Registration failed. Please try again.';
          this.cdr.detectChanges();
        }
      });
  }
}

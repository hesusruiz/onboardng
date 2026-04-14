import { Component, ViewEncapsulation, inject, ChangeDetectorRef } from '@angular/core';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { latinOnly } from '../validators/latin-only.validator';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { finalize } from 'rxjs';
import { ProviderLandingComponent } from './components/landing/provider-landing.component';
import { TermsComponent } from '../components/terms/terms.component';
import { RepresentativeComponent } from '../components/representative/representative.component';
import { CompanyInfoComponent } from '../components/company-info/company-info.component';
import { EmailVerificationComponent } from '../components/email-verification/email-verification.component';
import { ROLES } from '../constants';
import { ProviderSuccessComponent } from './components/success/provider-success.component';
import { RegistrationContextService } from '../services/registration-context.service';

@Component({
  selector: 'app-onboarding-provider',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    ProviderLandingComponent,
    TermsComponent,
    RepresentativeComponent,
    CompanyInfoComponent,
    EmailVerificationComponent,
    ProviderSuccessComponent
  ],
  templateUrl: './onboarding-provider.component.html',
  styleUrl: './onboarding-provider.component.css',
  encapsulation: ViewEncapsulation.None
})
export class OnboardingProviderComponent {
  private http = inject(HttpClient);
  private cdr = inject(ChangeDetectorRef);
  private registrationContext = inject(RegistrationContextService);

  currentStep = 0;
  showVerificationModal = false;
  showAlreadyRegisteredModal = false;
  showCompanyRegisteredModal = false;
  isValidatingEmail = false;
  emailError = '';
  isVerifyingCode = false;
  verifyError = '';
  isRegistering = false;
  registerError = '';
  verifiedCode = '';

  onboardingForm = new FormGroup({
    meetsRequirements: new FormControl(false, [Validators.requiredTrue]),
    acceptTerms: new FormControl(false, [Validators.requiredTrue]),
    representative: new FormGroup({
      firstName: new FormControl('', [Validators.required, latinOnly]),
      lastName: new FormControl('', [Validators.required, latinOnly]),
      email: new FormControl('', [Validators.required, Validators.email]),
      isAuthorised: new FormControl(false, [Validators.requiredTrue]),
    }),
    company: new FormGroup({
      legalName: new FormControl('', [Validators.required, latinOnly]),
      vatNumber: new FormControl('', [Validators.required, latinOnly]),
      country: new FormControl('', [Validators.required]),
      city: new FormControl('', [Validators.required, latinOnly]),
      street: new FormControl('', [Validators.required, latinOnly]),
      postalCode: new FormControl('', [Validators.required, latinOnly]),
    }),
  });

  get meetsRequirementsControl(): FormControl {
    return this.onboardingForm.get('meetsRequirements') as FormControl;
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

  goBackToRepresentative(): void {
    this.verifiedCode = '';
    this.currentStep = 2;
    this.isRegistering = false;
    this.registerError = '';
    this.showCompanyRegisteredModal = false;
    this.showAlreadyRegisteredModal = false;
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
      .pipe(finalize(() => {
        this.isValidatingEmail = false;
        this.cdr.detectChanges();
      }))
      .subscribe({
        next: (res: any) => {
          this.isValidatingEmail = false;
          if (res && res.success === true) {
            this.showVerificationModal = true;
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

  onEditEmail(): void {
    this.showVerificationModal = false;
    this.verifyError = '';
    this.isValidatingEmail = false;
    this.emailError = '';
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
            this.showVerificationModal = false;
            const d = res.data;
            this.companyGroup.patchValue({
              legalName: d?.companyName || '',
              vatNumber: d?.vatId || '',
              country: d?.country || '',
              city: d?.city || '',
              street: d?.streetAddress || '',
              postalCode: d?.postalCode || '',
            });
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
      city: company.city,
      streetAddress: company.street,
      postalCode: company.postalCode,
      code: this.verifiedCode,
      role: ROLES.SELLER,
    };

    this.http.post<any>('/api/register', body, { headers })
      .pipe(finalize(() => {
        this.isRegistering = false;
        this.cdr.detectChanges();
      }))
      .subscribe({
        next: (res: any) => {
          this.isRegistering = false;
          if (res && res.success === true) {
            this.registrationContext.setContext({
              vatId: company.vatNumber,
              email: rep.email,
              code: this.verifiedCode
            });
            this.currentStep = 5;
          } else {
            this.registerError = res?.message || 'Registration failed. Please try again.';
          }
          this.cdr.detectChanges();
        },
        error: (err: any) => {
          this.isRegistering = false;
          const message: string = err.error?.message || '';
          if (message.toLowerCase().includes('email already registered')) {
            this.showAlreadyRegisteredModal = true;
          } else if (message.toLowerCase().includes('company already registered')) {
            this.showCompanyRegisteredModal = true;
          } else {
            this.registerError = message || 'Registration failed. Please try again.';
          }
          this.cdr.detectChanges();
        }
      });
  }
}

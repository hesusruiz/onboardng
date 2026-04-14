import { Component, ViewEncapsulation } from '@angular/core';
import { Router } from '@angular/router';

@Component({
  selector: 'app-provider-success',
  standalone: true,
  templateUrl: './provider-success.component.html',
  styleUrl: './provider-success.component.css',
  encapsulation: ViewEncapsulation.None
})
export class ProviderSuccessComponent {
  constructor(private router: Router) {}

  completeOnboarding(): void {
    this.router.navigate(['/onboard-provider']);
  }

  goHome(): void {
    window.location.href = 'https://dome-marketplace-sbx.org/';
  }
}

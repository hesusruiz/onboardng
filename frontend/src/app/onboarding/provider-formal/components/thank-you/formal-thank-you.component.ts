import { Component, ViewEncapsulation } from '@angular/core';
import { Router } from '@angular/router';

@Component({
  selector: 'app-formal-thank-you',
  standalone: true,
  templateUrl: './formal-thank-you.component.html',
  styleUrl: './formal-thank-you.component.css',
  encapsulation: ViewEncapsulation.None
})
export class FormalThankYouComponent {
  constructor(private router: Router) {}

  goHome(): void {
    this.router.navigate(['/dashboard']);
  }
}

import { Component, ViewEncapsulation } from '@angular/core';
import { Router } from '@angular/router';

@Component({
  selector: 'app-success',
  standalone: true,
  templateUrl: './success.component.html',
  styleUrl: './success.component.css',
  encapsulation: ViewEncapsulation.None
})
export class SuccessComponent {
  constructor(private router: Router) {}

  goHome(): void {
    this.router.navigate(['/dashboard']);
  }
}

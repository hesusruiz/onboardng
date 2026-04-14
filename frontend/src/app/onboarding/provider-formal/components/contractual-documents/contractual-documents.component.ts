import { Component, ViewEncapsulation, EventEmitter, Input, Output } from '@angular/core';
import { FormGroup, ReactiveFormsModule } from '@angular/forms';
import { NgClass } from '@angular/common';
import { ProgressBarComponent } from '../../../components/progress-bar/progress-bar.component';
import { StepFooterComponent } from '../../../components/step-footer/step-footer.component';

@Component({
  selector: 'app-contractual-documents',
  standalone: true,
  imports: [ReactiveFormsModule, NgClass, ProgressBarComponent, StepFooterComponent],
  templateUrl: './contractual-documents.component.html',
  styleUrl: './contractual-documents.component.css',
  encapsulation: ViewEncapsulation.None
})
export class ContractualDocumentsComponent {
  @Input() formGroup!: FormGroup;
  @Output() complete = new EventEmitter<void>();
  @Output() back = new EventEmitter<void>();

  isDragging = false;
  uploadedFiles: File[] = [];

  get hasEidas(): boolean | null {
    return this.formGroup.get('hasEidasCertificate')?.value;
  }

  downloadLearAppointment(): void {
    const a = document.createElement('a');
    a.href = 'CSP Account administrator appointment Form v.March 2026.docx';
    a.download = 'CSP Account administrator appointment Form v.March 2026.docx';
    a.click();
  }

  onDragOver(event: DragEvent): void {
    event.preventDefault();
    this.isDragging = true;
  }

  onDragLeave(): void {
    this.isDragging = false;
  }

  onDrop(event: DragEvent): void {
    event.preventDefault();
    this.isDragging = false;
    const files = event.dataTransfer?.files;
    if (files) {
      this.addFiles(files);
    }
  }

  onFileSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (input.files) {
      this.addFiles(input.files);
      input.value = '';
    }
  }

  removeFile(index: number): void {
    this.uploadedFiles.splice(index, 1);
  }

  onComplete(): void {
    if (this.formGroup.get('hasEidasCertificate')?.value === null) {
      this.formGroup.get('hasEidasCertificate')?.markAsTouched();
      return;
    }
    this.complete.emit();
  }

  formatFileSize(bytes: number): string {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / 1048576).toFixed(1) + ' MB';
  }

  private readonly allowedTypes = [
    'application/pdf',
    'application/msword',
    'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
  ];

  private addFiles(fileList: FileList): void {
    const filtered = Array.from(fileList).filter(f => this.allowedTypes.includes(f.type));
    this.uploadedFiles.push(...filtered);
  }
}

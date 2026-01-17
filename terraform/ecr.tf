resource "aws_ecr_repository" "ingestion" {
  name                 = "${var.project_name}-ingestion"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = {
    Name = "${var.project_name}-ingestion"
  }
}

resource "aws_ecr_lifecycle_policy" "ingestion" {
  repository = aws_ecr_repository.ingestion.name

  policy = jsonencode({
    rules = [
      {
        rulePriority = 1
        description  = "Keep last 10 images"
        selection = {
          tagStatus   = "any"
          countType   = "imageCountMoreThan"
          countNumber = 10
        }
        action = {
          type = "expire"
        }
      }
    ]
  })
}

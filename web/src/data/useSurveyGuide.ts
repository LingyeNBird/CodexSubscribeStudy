import { computed, nextTick, ref, watch, type ComputedRef } from "vue";

export interface SurveyQuestion {
  id: string;
  title: string;
  section: string;
  answer: string;
  when?: boolean;
  error?: string;
}

export function useSurveyGuide(questions: ComputedRef<SurveyQuestion[]>) {
  const form = ref<HTMLFormElement>();
  const guided = ref(true);
  try {
    guided.value =
      localStorage.getItem("chatgpt-account-survey:mode") !== "form";
  } catch {
    // Mode switching also works when browser storage is unavailable.
  }
  const currentId = ref("status");
  const error = ref("");
  const reviewing = computed(() => currentId.value === "review");
  const currentIndex = computed(() =>
    questions.value.findIndex((item) => item.id === currentId.value),
  );
  const current = computed(() => questions.value[currentIndex.value]);
  const answered = computed(() =>
    questions.value.filter((item) => item.answer),
  );
  const progress = computed(() =>
    reviewing.value ? questions.value.length : currentIndex.value + 1,
  );

  function element(id: string) {
    return form.value?.querySelector<HTMLElement>(`[data-question="${id}"]`);
  }
  async function focusCurrent() {
    await nextTick();
    const target = element(currentId.value);
    target?.focus({ preventScroll: true });
    const scrollTarget = guided.value ? form.value : target;
    scrollTarget?.scrollIntoView({
      block: guided.value ? "start" : "center",
      behavior: "instant",
    });
  }
  async function go(id: string) {
    currentId.value = id;
    error.value = "";
    await focusCurrent();
  }
  function show(id: string) {
    return (
      questions.value.some((item) => item.id === id) &&
      (!guided.value || currentId.value === id)
    );
  }
  function showSection(section: string) {
    return !guided.value || current.value?.section === section;
  }
  async function setMode(value: boolean) {
    if (value === guided.value) return;
    if (value && !reviewing.value) {
      const firstUnanswered = questions.value.find((item) => !item.answer);
      if (!current.value) currentId.value = firstUnanswered?.id ?? "review";
    }
    guided.value = value;
    error.value = "";
    try {
      localStorage.setItem(
        "chatgpt-account-survey:mode",
        value ? "guided" : "form",
      );
    } catch {
      // The draft is held in the form, not in browser storage.
    }
    await focusCurrent();
  }
  function trackQuestion(event: FocusEvent) {
    if (guided.value || !(event.target instanceof Element)) return;
    const id =
      event.target.closest<HTMLElement>("[data-question]")?.dataset.question;
    if (id && id !== "review") currentId.value = id;
  }
  async function validate(question: SurveyQuestion) {
    const invalid = Array.from(
      element(question.id)?.querySelectorAll<
        HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement
      >("input, select, textarea") ?? [],
    ).find((input) => input.willValidate && !input.validity.valid);
    if (!question.error && !invalid) return true;
    await go(question.id);
    error.value = question.error ?? "请检查这一题的填写内容。";
    if (invalid) invalid.reportValidity();
    return false;
  }
  async function next() {
    const question = current.value;
    if (!question || !(await validate(question)) || currentId.value !== question.id) return;
    await go(questions.value[currentIndex.value + 1]?.id ?? "review");
  }
  function previous() {
    const index = reviewing.value ? questions.value.length : currentIndex.value;
    return go(questions.value[Math.max(0, index - 1)]!.id);
  }
  async function validateAll() {
    for (const question of questions.value) {
      if (!(await validate(question))) return false;
    }
    return true;
  }
  watch(questions, (nextQuestions, previousQuestions) => {
    if (
      reviewing.value ||
      nextQuestions.some((item) => item.id === currentId.value)
    )
      return;
    const previousIndex = previousQuestions.findIndex(
      (item) => item.id === currentId.value,
    );
    currentId.value =
      previousQuestions
        .slice(0, previousIndex)
        .reverse()
        .find((item) => nextQuestions.some((next) => next.id === item.id))
        ?.id ?? nextQuestions[0]!.id;
  });
  return {
    form,
    guided,
    currentId,
    current,
    currentIndex,
    reviewing,
    answered,
    progress,
    error,
    show,
    showSection,
    setMode,
    trackQuestion,
    go,
    next,
    previous,
    validateAll,
  };
}

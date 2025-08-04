<script lang="ts">
  import { mdiFolder, mdiHelpCircle, mdiCheck } from '@mdi/js';
  import { getTranslate } from '@tolgee/svelte';

  import SvgIcon from '$lib/components/SVGIcon.svelte';
  import T from '$lib/components/T.svelte';
  import { getModalStore } from '$lib/skeletonExtensions';
  import { selectedProfile } from '$lib/store/ficsitCLIStore';
  import { error } from '$lib/store/generalStore';
  import { OpenFileDialog } from '$wailsjs/go/app/app';
  import { AddProfile, SelectInstall } from '$wailsjs/go/ficsitcli/ficsitCLI';
  import { SetCrackedInstallSetupComplete } from '$wailsjs/go/settings/settings';
  import { AddCustomInstallation } from '$wailsjs/go/app/app';

  export let parent: { onClose: () => void };

  const { t } = getTranslate();
  const modalStore = getModalStore();

  let step = 1; // 1: initial question, 2: directory selection, 3: success
  let customInstallDir = '';
  let customInstallDirError = '';
  let isProcessing = false;

  async function selectCustomInstallDir() {
    try {
      const result = await OpenFileDialog({
        title: "Select Satisfactory Game Directory",
        filters: [],
        canCreateDirectories: true,
        resolvesAliases: true,
        treatPackagesAsDirectories: true,
        showHiddenFiles: true,
      });
      
      if (result) {
        // Clean up the path to make sure we have the directory containing the game files
        let cleanPath = result;
        if (cleanPath.endsWith('.exe')) {
          // If user selected an executable, go to its parent directory
          cleanPath = cleanPath.substring(0, cleanPath.lastIndexOf('\\'));
        }
        customInstallDir = cleanPath;
        customInstallDirError = '';
      }
    } catch (err) {
      customInstallDirError = 'Failed to select directory: ' + (err as string);
    }
  }

  async function setupCrackedInstall() {
    if (!customInstallDir) {
      customInstallDirError = 'Please select a directory';
      return;
    }
    
    isProcessing = true;
    try {
      // Add the custom installation
      const install = await AddCustomInstallation(customInstallDir);
      
      // Select the installation
      await SelectInstall(install.path);
      
      // Create the "Custom" profile
      await AddProfile('Custom');
      await selectedProfile.asyncSet('Custom');
      
      // Mark the cracked install setup as complete
      await SetCrackedInstallSetupComplete(true);
      
      step = 3;
    } catch (err) {
      customInstallDirError = 'Failed to set up cracked installation. Make sure you selected the correct Satisfactory installation folder (the one containing FactoryGame.exe). Error: ' + (err as string);
      error.set(customInstallDirError);
    } finally {
      isProcessing = false;
    }
  }

  function skipSetup() {
    SetCrackedInstallSetupComplete(true);
    parent.onClose();
  }

  function finishSetup() {
    parent.onClose();
  }
</script>

<div
  style="max-height: calc(100vh - 3rem); max-width: calc(100vw - 3rem);"
  class="w-[48rem] card flex flex-col gap-6"
>
  {#if step === 1}
    <header class="card-header font-bold text-2xl text-center">
      <T defaultValue="Installation Setup" keyName="cracked_install_setup.title" />
    </header>
    <section class="px-4">
      <div class="flex items-center gap-4 mb-4">
        <span class="badge bg-warning-500">
          <SvgIcon class="h-6 w-6 my-1" icon={mdiHelpCircle} />
        </span>
        <div>
          <p class="text-lg font-semibold">
            <T
              defaultValue="Do you have a cracked version of Satisfactory?"
              keyName="cracked_install_setup.question"
            />
          </p>
          <p class="text-sm text-surface-600-300-token mt-1">
            <T
              defaultValue="If you're using a cracked version, we can help you set up a custom installation directory and create a dedicated profile."
              keyName="cracked_install_setup.description"
            />
          </p>
        </div>
      </div>
    </section>
    <footer class="card-footer">
      <button class="btn variant-ringed" on:click={skipSetup}>
        <T defaultValue="No, I have the official version" keyName="cracked_install_setup.no" />
      </button>
      <button class="btn variant-filled-primary" on:click={() => step = 2}>
        <T defaultValue="Yes, help me set it up" keyName="cracked_install_setup.yes" />
      </button>
    </footer>
  {:else if step === 2}
    <header class="card-header font-bold text-2xl text-center">
      <T defaultValue="Select Satisfactory Installation" keyName="cracked_install_setup.select_title" />
    </header>
    <section class="px-4">
      <p class="text-base mb-4">
        <T
          defaultValue="Please select your Satisfactory installation directory. This should be the folder containing FactoryGame.exe."
          keyName="cracked_install_setup.select_description"
        />
      </p>
      
      <div class="space-y-4">
        <div class="flex items-center space-x-2">
          <input 
            type="text" 
            class="input px-4 py-2 flex-1" 
            bind:value={customInstallDir}
            placeholder="Select installation directory..."
            readonly
          >
          <button 
            class="btn variant-filled"
            on:click={selectCustomInstallDir}
            disabled={isProcessing}
          >
            <SvgIcon class="h-5 w-5 mr-2" icon={mdiFolder} />
            <T defaultValue="Browse" keyName="common.browse" />
          </button>
        </div>
        
        {#if customInstallDirError}
          <div class="text-error-500">
            {customInstallDirError}
          </div>
        {/if}
      </div>
    </section>
    <footer class="card-footer">
      <button class="btn variant-ringed" on:click={() => step = 1} disabled={isProcessing}>
        <T defaultValue="Back" keyName="common.back" />
      </button>
      <button
        class="btn variant-filled-primary"
        on:click={setupCrackedInstall}
        disabled={!customInstallDir || isProcessing}
      >
        {#if isProcessing}
          <span class="animate-spin mr-2">⏳</span>
        {/if}
        <T defaultValue="Set Up Custom Installation" keyName="cracked_install_setup.setup" />
      </button>
    </footer>
  {:else if step === 3}
    <header class="card-header font-bold text-2xl text-center text-success-500">
      <SvgIcon class="h-8 w-8 mx-auto mb-2" icon={mdiCheck} />
      <T defaultValue="Setup Complete!" keyName="cracked_install_setup.success_title" />
    </header>
    <section class="px-4">
      <p class="text-base text-center">
        <T
          defaultValue="Your cracked Satisfactory installation has been set up successfully! A 'Custom' profile has been created for you."
          keyName="cracked_install_setup.success_description"
        />
      </p>
    </section>
    <footer class="card-footer">
      <button class="btn variant-filled-primary w-full" on:click={finishSetup}>
        <T defaultValue="Get Started!" keyName="cracked_install_setup.finish" />
      </button>
    </footer>
  {/if}
</div>